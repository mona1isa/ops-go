#!/bin/sh
# 增量读取远端日志文件，供 Ops-Go 堡垒机通道下发执行。
# 用法: tail.sh <file> <last_pos> <last_inode> <max_bytes> <initial_back_bytes>
# 输出: 首行 OPS_META 元信息，其后为原始日志内容（不做转义，避免依赖 jq/python）
# 失败: 输出一行 OPS_ERR 并以非 0 退出

FILE="$1"
POS="${2:-0}"
LAST_INODE="${3:-0}"
MAX_BYTES="${4:-524288}"
INITIAL_BACK="${5:-65536}"

if [ -z "$FILE" ]; then
  echo "OPS_ERR 未指定日志文件路径"
  exit 2
fi

# 通配符路径：取最近修改的匹配文件（日志轮转场景下自动跟随新文件）
case "$FILE" in
*'*'*|*'?'*|*'['*)
  MATCHED=""
  NEWEST=0
  for candidate in $FILE; do
    if [ -f "$candidate" ]; then
      TS=$(stat -c '%Y' "$candidate" 2>/dev/null)
      case "$TS" in ''|*[!0-9]*) TS=0 ;; esac
      if [ "$TS" -ge "$NEWEST" ]; then
        NEWEST=$TS
        MATCHED=$candidate
      fi
    fi
  done
  if [ -z "$MATCHED" ]; then
    echo "OPS_ERR 通配符未匹配到日志文件: $FILE"
    exit 3
  fi
  FILE="$MATCHED"
  ;;
esac

if [ ! -f "$FILE" ]; then
  echo "OPS_ERR 日志文件不存在: $FILE"
  exit 3
fi

if [ ! -r "$FILE" ]; then
  echo "OPS_ERR 日志文件不可读(需要 root 或加入 adm 组): $FILE"
  exit 4
fi

INFO=$(stat -c '%i %s' "$FILE" 2>/dev/null)
if [ -z "$INFO" ]; then
  echo "OPS_ERR 无法获取文件状态(stat)"
  exit 5
fi
INODE=$(echo "$INFO" | cut -d' ' -f1)
SIZE=$(echo "$INFO" | cut -d' ' -f2)

# 参数非数字时归零，避免 shell 算术报错
case "$POS" in ''|*[!0-9]*) POS=0 ;; esac
case "$SIZE" in ''|*[!0-9]*) SIZE=0 ;; esac
case "$MAX_BYTES" in ''|*[!0-9]*) MAX_BYTES=524288 ;; esac
case "$INITIAL_BACK" in ''|*[!0-9]*) INITIAL_BACK=0 ;; esac

# 首次采集或文件轮转/截断：重新定位读取起点
if [ "$INODE" != "$LAST_INODE" ] || [ "$SIZE" -lt "$POS" ]; then
  if [ "$LAST_INODE" = "0" ] && [ "$INITIAL_BACK" -gt 0 ] && [ "$SIZE" -gt "$INITIAL_BACK" ]; then
    # 首次接入只回溯尾部内容，不重复拉取历史全量
    POS=$((SIZE - INITIAL_BACK))
  else
    POS=0
  fi
fi

AVAIL=$((SIZE - POS))
DROPPED=0
if [ "$MAX_BYTES" -gt 0 ] && [ "$AVAIL" -gt "$MAX_BYTES" ]; then
  # 单轮读取上限保护：丢弃最旧的部分并前进偏移量，保证不因积压卡死
  DROPPED=$((AVAIL - MAX_BYTES))
  POS=$((POS + DROPPED))
  AVAIL=$MAX_BYTES
fi

NEW_POS=$((POS + AVAIL))
echo "OPS_META inode=$INODE pos=$NEW_POS size=$SIZE dropped=$DROPPED file=$FILE"

if [ "$AVAIL" -gt 0 ]; then
  tail -c +"$((POS + 1))" "$FILE" 2>/dev/null | head -c "$AVAIL"
fi

exit 0
