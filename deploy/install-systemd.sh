#!/usr/bin/env bash
# 安装/更新 ops-go 的 systemd 服务。装好之后，更新服务就用：
#
#   sudo systemctl restart ops-go
#   或者 ./restart.sh          （脚本检测到单元存在会自动走 systemctl）
#
# 用法：
#   ./deploy/install-systemd.sh              # 构建 + 装单元 + 立即接管 + 开机自启
#   ./deploy/install-systemd.sh --no-enable  # 同上，但不设置开机自启
#   ./deploy/install-systemd.sh --status     # 只看状态，不做任何改动
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

SERVICE=ops-go
UNIT_SRC="$PROJECT_DIR/deploy/ops-go.service"
UNIT_DST="/etc/systemd/system/$SERVICE.service"
BINARY="$PROJECT_DIR/$SERVICE"
LOG_FILE="$PROJECT_DIR/logs/$SERVICE.log"

# 读 .env 里的键，缺失时返回空串：grep 无匹配会退出 1，配合 pipefail 会把整个脚本带崩，
# 必须在这里吞掉，让下面的默认值兜底
read_env() {
	grep -E "^$1=" "$PROJECT_DIR/.env" 2>/dev/null | head -n 1 | cut -d= -f2- | tr -d '"' | tr -d "'" || true
}

SSH_PORT="$(read_env SSH_PORT)"
SSH_PORT="${SSH_PORT:-2222}"
APP_PORT="$(read_env APP_PORT)"
APP_PORT="${APP_PORT:-8080}"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }
die() {
	echo "错误: $*" >&2
	exit 1
}

# 该 PID 是否已由 systemd 托管（cgroup v2 形如 0::/system.slice/ops-go.service）
managed_by_systemd() {
	grep -q "$SERVICE.service" "/proc/$1/cgroup" 2>/dev/null
}

# 停掉历史上 restart.sh 用 nohup 起的实例：只认工作目录在本项目、且不由 systemd 托管的进程，
# 避免误杀同名二进制，也避免和 systemd 抢着管同一个进程
stop_legacy_instance() {
	local pids="" pid cwd cmdline
	for pid in $(pgrep -x "$SERVICE" 2>/dev/null || true); do
		[[ -d "/proc/$pid" ]] || continue
		managed_by_systemd "$pid" && continue
		cwd="$(readlink -f "/proc/$pid/cwd" 2>/dev/null || true)"
		cmdline="$(tr '\0' ' ' <"/proc/$pid/cmdline" 2>/dev/null || true)"
		if [[ "$cwd" != "$PROJECT_DIR" && "$cmdline" != "$BINARY"* ]]; then
			log "跳过不属于本项目的进程 $pid（$cmdline）"
			continue
		fi
		log "停止旧的 nohup 实例 PID $pid"
		kill -TERM "$pid" 2>/dev/null || true
		pids+="$pid"$'\n'
	done

	local waited alive
	waited=0
	while [[ $waited -lt 30 ]]; do
		alive=0
		for pid in $pids; do
			[[ -d "/proc/$pid" ]] && alive=1
		done
		[[ $alive -eq 0 ]] && break
		sleep 1
		waited=$((waited + 1))
	done
	for pid in $pids; do
		if [[ -d "/proc/$pid" ]]; then
			log "PID $pid 未响应 SIGTERM，强制结束"
			kill -KILL "$pid" 2>/dev/null || true
		fi
	done
}

port_listening() {
	ss -ltnH "sport = :$1" 2>/dev/null | grep -q LISTEN
}

do_status() {
	if [[ ! -f "$UNIT_DST" ]]; then
		echo "未安装 systemd 单元：$UNIT_DST 不存在（服务可能仍由 restart.sh 以 nohup 方式运行）"
	else
		sudo systemctl status "$SERVICE" --no-pager || true
		echo "--- 开机自启: $(systemctl is-enabled "$SERVICE" 2>&1) ---"
		echo "--- 最近日志 ---"
		tail -n 15 "$LOG_FILE" 2>/dev/null || true
	fi
	echo "--- 监听端口 ---"
	if port_listening "$APP_PORT" || port_listening "$SSH_PORT"; then
		ss -ltnp 2>/dev/null | grep -E ":($APP_PORT|$SSH_PORT)\b" || true
	else
		echo "（$APP_PORT/$SSH_PORT 均未监听）"
	fi
}

case "${1:-install}" in
--status | status)
	do_status
	exit 0
	;;
install | --no-enable)
	;;
*)
	echo "用法: $0 [--no-enable | --status]" >&2
	exit 1
	;;
esac

[[ -f "$UNIT_SRC" ]] || die "找不到单元模板 $UNIT_SRC"
sudo -n true 2>/dev/null || die "需要 sudo 权限写 /etc/systemd/system（可在会话里用 ! 前缀交互执行）"

log "构建最新二进制"
go build -o "$BINARY" .

[[ -f "$UNIT_DST" ]] && log "更新已安装的单元文件" || log "首次安装单元文件"
sudo cp "$UNIT_SRC" "$UNIT_DST"
sudo systemctl daemon-reload

# 旧实例还占着端口时，systemd 拉起来的新实例会因 bind 失败而反复重启，所以先清场
stop_legacy_instance

log "通过 systemd 启动 $SERVICE"
sudo systemctl restart "$SERVICE"

waited=0
until port_listening "$APP_PORT"; do
	sleep 1
	waited=$((waited + 1))
	if [[ $waited -ge 30 ]]; then
		echo "启动超时，服务状态："
		sudo systemctl status "$SERVICE" --no-pager || true
		tail -n 20 "$LOG_FILE" 2>/dev/null || true
		exit 1
	fi
done

if [[ "${1:-}" == "--no-enable" ]]; then
	log "已接管运行，未设置开机自启（需要时：sudo systemctl enable $SERVICE）"
else
	sudo systemctl enable "$SERVICE" >/dev/null
	log "已设置开机自启"
fi

log "服务已就绪（API :$APP_PORT，堡垒机 :$SSH_PORT）"
do_status
