#!/usr/bin/env bash
# ops-go 后端服务更新脚本：停止旧进程 -> 重新构建 -> 重启服务
#
# 用法：
#   ./restart.sh          # 停止 -> 构建 -> 启动
#   ./restart.sh stop     # 仅停止服务
#   ./restart.sh start    # 仅启动服务（不做构建）
#
# 注意：必须在 ops-go 目录下执行，godotenv.Load() 只会从当前工作目录读取 .env

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

BINARY="ops-go"
LOG_DIR="$PROJECT_DIR/logs"
LOG_FILE="$LOG_DIR/ops-go.log"
WAIT_TIMEOUT=30

# 读取服务端口（用于查找旧进程和启动健康检查）
APP_PORT="$(grep -E '^APP_PORT=' .env 2>/dev/null | cut -d= -f2 | tr -d '"' | tr -d "'")"
APP_PORT="${APP_PORT:-8080}"

log() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# 判断进程是否属于本项目（按命令行或工作目录匹配）
is_project_process() {
	local pid="$1"
	local cwd cmd
	cwd="$(readlink -f "/proc/$pid/cwd" 2>/dev/null || true)"
	cmd="$(tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null || true)"
	[[ "$cwd" == "$PROJECT_DIR" ]] && return 0
	[[ "$cmd" == *"$PROJECT_DIR/$BINARY"* ]] && return 0
	return 1
}

# 收集需要停止的进程：监听 APP_PORT 的进程 + go run/二进制进程
collect_pids() {
	local pids=""
	local pid

	# 1. 占用服务端口的进程
	if command -v ss >/dev/null 2>&1; then
		pids+="$(ss -ltnpH "sport = :$APP_PORT" 2>/dev/null | grep -oE 'pid=[0-9]+' | cut -d= -f2 || true)"
		pids+=$'\n'
	fi
	# 2. go run main.go 及其编译产物进程
	for pattern in 'go run main\.go' 'go-build.*-d/main' "^$PROJECT_DIR/$BINARY\$"; do
		pids+="$(pgrep -f "$pattern" 2>/dev/null || true)"
		pids+=$'\n'
	done

	# 去重并过滤掉不属于本项目的进程
	echo "$pids" | tr ' ' '\n' | grep -E '^[0-9]+$' | sort -un | while read -r pid; do
		# 端口占用的进程可能来自其他目录，这里只过滤明显无关的
		if [[ "$pid" == "$$" ]]; then
			continue
		fi
		echo "$pid"
	done
}

stop_service() {
	local pids pid waited
	pids="$(collect_pids)"
	if [[ -z "$pids" ]]; then
		log "没有发现运行中的服务进程"
		return 0
	fi

	for pid in $pids; do
		if [[ ! -d "/proc/$pid" ]]; then
			continue
		fi
		if ! is_project_process "$pid"; then
			log "跳过不属于本项目的进程 $pid"
			continue
		fi
		log "停止进程 $pid ($(tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null | cut -c1-80))"
		kill -TERM "$pid" 2>/dev/null || true
	done

	# 等待进程退出，超时后强制结束
	waited=0
	while [[ $waited -lt $WAIT_TIMEOUT ]]; do
		local alive=0
		for pid in $pids; do
			[[ -d "/proc/$pid" ]] && alive=1
		done
		[[ $alive -eq 0 ]] && break
		sleep 1
		waited=$((waited + 1))
	done

	for pid in $pids; do
		if [[ -d "/proc/$pid" ]]; then
			log "进程 $pid 未正常退出，强制结束"
			kill -KILL "$pid" 2>/dev/null || true
		fi
	done
	sleep 1
	log "服务已停止"
}

build_service() {
	log "开始构建：$PROJECT_DIR/$BINARY"
	go build -o "$BINARY" .
	log "构建完成"
}

start_service() {
	mkdir -p "$LOG_DIR"
	log "启动服务，端口：$APP_PORT，日志：$LOG_FILE"
	nohup "./$BINARY" >>"$LOG_FILE" 2>&1 &

	# 等待端口监听
	local waited=0
	while [[ $waited -lt $WAIT_TIMEOUT ]]; do
		if command -v ss >/dev/null 2>&1 && ss -ltnH "sport = :$APP_PORT" 2>/dev/null | grep -q LISTEN; then
			log "服务启动成功（PID $(pgrep -f "^$PROJECT_DIR/$BINARY\$" | tr '\n' ' ')）"
			return 0
		fi
		sleep 1
		waited=$((waited + 1))
	done

	log "服务启动超时，请检查日志：$LOG_FILE"
	tail -n 20 "$LOG_FILE" || true
	exit 1
}

case "${1:-restart}" in
stop)
	stop_service
	;;
start)
	start_service
	;;
restart)
	stop_service
	build_service
	start_service
	;;
*)
	echo "用法: $0 [restart|stop|start]"
	exit 1
	;;
esac
