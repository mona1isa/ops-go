#!/bin/bash
# Ops-Go 主机指标采集脚本
# 由监控采集服务通过 SSH 下发执行，stdout 只输出一个 JSON 对象。
# 依赖：/proc、df、awk、sed —— 仅做只读采集，除临时采样文件外不写任何位置。

# 丢弃诊断输出，保证 stdout 为纯净 JSON（SSH 侧用 CombinedOutput 合并了 stderr）
exec 2>/dev/null

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/ops_metric_XXXXXX") || exit 1
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

fail() {
    printf '{"error":"%s"}\n' "$1"
    exit 1
}

# 把 TSV 行拼成 JSON 对象数组内容，字段名由调用方给出
# 用法：awk ... | to_json mountpoint fstype used_percent
to_json() {
    awk -F '\t' -v f1="$1" -v f2="$2" -v f3="$3" '
        function emit(name, val,   isnum) {
            isnum = (val ~ /^-?[0-9]+(\.[0-9]+)?$/)
            if (isnum) return sprintf("\"%s\":%s", name, val)
            return sprintf("\"%s\":\"%s\"", name, val)
        }
        {
            sep = (NR == 1 ? "{" : ",{")
            printf "%s%s,%s,%s}", sep, emit(f1, $1), emit(f2, $2), emit(f3, $3)
        }'
}

# 转义 JSON 字符串中的反斜杠与双引号
json_escape() {
    printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

KERNEL=$(uname -s)

case "$KERNEL" in
Linux)
    # ---- 第一次采样 ----
    awk '/^cpu / { total = 0; for (i = 2; i <= NF; i++) total += $i; print total, $5 + $6 }' /proc/stat > "$TMP_DIR/cpu1"
    if [ -r /proc/net/dev ]; then
        awk 'NR > 2 {
                split($1, a, ":"); dev = a[1]; gsub(/[ \t]/, "", dev)
                if (dev == "" || dev == "lo") next
                # 容器网络产生的临时网卡基数大且无监控价值
                if (dev ~ /^veth/ || dev ~ /^br-/ || dev ~ /^cni/) next
                print dev, $2, $10
             }' /proc/net/dev > "$TMP_DIR/nic1"
    fi

    # 采集窗口 1 秒，用于 CPU 使用率与网卡速率
    sleep 1

    # ---- 第二次采样 ----
    awk '/^cpu / { total = 0; for (i = 2; i <= NF; i++) total += $i; print total, $5 + $6 }' /proc/stat > "$TMP_DIR/cpu2"
    if [ -r /proc/net/dev ]; then
        awk 'NR > 2 {
                split($1, a, ":"); dev = a[1]; gsub(/[ \t]/, "", dev)
                if (dev == "" || dev == "lo") next
                if (dev ~ /^veth/ || dev ~ /^br-/ || dev ~ /^cni/) next
                print dev, $2, $10
             }' /proc/net/dev > "$TMP_DIR/nic2"
    fi

    set -- $(cat "$TMP_DIR/cpu1"); CPU_TOTAL1=$1; CPU_IDLE1=$2
    set -- $(cat "$TMP_DIR/cpu2"); CPU_TOTAL2=$1; CPU_IDLE2=$2

    if [ -z "$CPU_TOTAL1" ] || [ -z "$CPU_TOTAL2" ]; then
        fail "cpu_stat_unavailable"
    fi
    CPU_PERCENT=$(awk -v t1="$CPU_TOTAL1" -v i1="$CPU_IDLE1" -v t2="$CPU_TOTAL2" -v i2="$CPU_IDLE2" '
        BEGIN { dt = t2 - t1; di = i2 - i1
                if (dt <= 0) { print "0.00"; exit }
                if (di < 0) di = 0
                printf "%.2f", (dt - di) * 100 / dt }')

    # 内存：优先 MemAvailable，老内核退化到 MemFree+Buffers+Cached
    MEM_TOTAL=$(awk '/^MemTotal:/ { print $2 }' /proc/meminfo)
    MEM_AVAIL=$(awk '/^MemAvailable:/ { print $2 }' /proc/meminfo)
    if [ -z "$MEM_AVAIL" ]; then
        MEM_AVAIL=$(awk '/^MemFree:/ { f = $2 } /^Buffers:/ { b = $2 } /^Cached:/ { c = $2 } END { print f + b + c }' /proc/meminfo)
    fi
    if [ -z "$MEM_TOTAL" ] || [ "$MEM_TOTAL" -le 0 ] 2>/dev/null; then
        fail "meminfo_unavailable"
    fi
    MEM_PERCENT=$(awk -v t="$MEM_TOTAL" -v a="$MEM_AVAIL" 'BEGIN { printf "%.2f", (t - a) * 100 / t }')

    LOAD1=$(awk '{ print $1 }' /proc/loadavg)
    UPTIME=$(awk '{ printf "%.0f", $1 }' /proc/uptime)

    DISKS=$(df -T -P | awk '
        NR > 1 {
            fstype = $2; mount = $7; pct = $6; gsub(/%/, "", pct)
            if (pct + 0 != pct) next
            if (fstype == "tmpfs" || fstype == "devtmpfs" || fstype == "squashfs" || fstype == "overlay" ||
                fstype == "iso9660" || fstype == "autofs" || fstype == "ramfs" || fstype == "swap" ||
                fstype == "cgroup" || fstype == "cgroup2" || fstype == "pstore" || fstype == "hugetlbfs" ||
                fstype == "binfmt_misc" || fstype == "debugfs" || fstype == "tracefs" || fstype == "securityfs" ||
                fstype == "configfs" || fstype == "fusectl" || fstype == "mqueue" || fstype == "devpts" ||
                fstype == "proc" || fstype == "sysfs" || fstype == "nsfs" || fstype == "fuse.lxcfs") next
            if (mount ~ /^\/(proc|sys|dev|run)/) next
            gsub(/\\/, "\\\\\\\\", mount); gsub(/"/, "\\\"", mount)
            gsub(/\\/, "\\\\\\\\", fstype); gsub(/"/, "\\\"", fstype)
            printf "%s\t%s\t%s\n", mount, fstype, pct
        }' | to_json mountpoint fstype used_percent)

    if [ -s "$TMP_DIR/nic1" ] && [ -s "$TMP_DIR/nic2" ]; then
        NETS=$(awk '
            FNR == 1 { fi++ }
            { dev = $1; rx[fi "|" dev] = $2; tx[fi "|" dev] = $3; seen[dev] = 1 }
            END {
                for (dev in seen) {
                    r = rx["2|" dev] - rx["1|" dev]
                    t = tx["2|" dev] - tx["1|" dev]
                    if (r < 0 || !(("1|" dev) in rx)) r = 0
                    if (t < 0 || !(("1|" dev) in tx)) t = 0
                    esc = dev; gsub(/\\/, "\\\\\\\\", esc); gsub(/"/, "\\\"", esc)
                    printf "%s\t%.0f\t%.0f\n", esc, r, t
                }
            }' "$TMP_DIR/nic1" "$TMP_DIR/nic2" | to_json device rx_bytes_per_sec tx_bytes_per_sec)
    else
        NETS=""
    fi
    ;;
Darwin)
    # macOS/BSD 精简分支：仅提供 CPU、内存、负载、运行时长与磁盘，网卡速率不采集
    CPU_LINE=$(top -l 2 -n 0 | awk '/CPU usage/ { line = $0 } END { print line }')
    [ -n "$CPU_LINE" ] || fail "unsupported_platform"
    CPU_PERCENT=$(printf '%s' "$CPU_LINE" | awk '{
        for (i = 1; i <= NF; i++) if ($i == "user") { v = $(i - 1); gsub(/%/, "", v); print v; exit }
    }')
    [ -n "$CPU_PERCENT" ] || CPU_PERCENT="0.00"

    LOAD1=$(sysctl -n vm.loadavg | awk '{ gsub(/[{}]/, ""); print $1 }')
    [ -n "$LOAD1" ] || LOAD1="0.00"

    BOOT=$(sysctl -n kern.boottime | sed -e 's/.*sec = \([0-9]*\).*/\1/')
    NOW=$(date +%s)
    [ -n "$BOOT" ] && UPTIME=$((NOW - BOOT)) || UPTIME=0

    MEM_TOTAL=$(sysctl -n hw.memsize)
    MEM_USED=$(vm_stat | awk '/Pages active/ { a = $3 } /Pages wired/ { w = $4 } /occupied by compressor/ { c = $5 }
        END { gsub(/\./, "", a); gsub(/\./, "", w); gsub(/\./, "", c); printf "%.0f", (a + w + c) * 4096 }')
    MEM_PERCENT=$(awk -v t="$MEM_TOTAL" -v u="$MEM_USED" 'BEGIN { if (t > 0 && u != "") printf "%.2f", u * 100 / t; else print "0.00" }')

    DISKS=$(df -P | awk '
        NR > 1 {
            pct = $5; gsub(/%/, "", pct); mount = $6; fs = $1
            if (fs !~ /^\/dev\/disk/ || pct + 0 != pct) next
            gsub(/\\/, "\\\\\\\\", mount); gsub(/"/, "\\\"", mount)
            printf "%s\tapfs\t%s\n", mount, pct
        }' | to_json mountpoint fstype used_percent)
    NETS=""
    ;;
*)
    fail "unsupported_platform"
    ;;
esac

[ -n "$LOAD1" ] || LOAD1="0.00"
[ -n "$UPTIME" ] || UPTIME=0

printf '{"cpu_percent":%s,"mem_used_percent":%s,"load1":%s,"uptime_seconds":%s,"os":"%s","disks":[%s],"nets":[%s]}\n' \
    "$CPU_PERCENT" "$MEM_PERCENT" "$LOAD1" "$UPTIME" "$(json_escape "$KERNEL")" "$DISKS" "$NETS"
