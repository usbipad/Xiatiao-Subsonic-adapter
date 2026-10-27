#!/bin/sh
# 桌面启动入口：后台运行 adapter，不依赖终端窗口存活。
#
# 设计目标：
#   1. 从桌面图标启动时不弹终端，关掉任何窗口都不影响服务；
#   2. 启动失败时用桌面通知提示用户，而不是留一个孤零零的终端窗口；
#   3. 日志写到文件，便于事后排查。
#
# 用法：
#   ./launch.sh              # 正常启动（已运行则只打开管理页）
#   ./launch.sh --foreground # 前台运行，用于调试
#   ./launch.sh --no-browser # 启动但不打开浏览器（供开机自启使用）

set -u

DIR=$(cd "$(dirname "$0")" && pwd)
cd "$DIR" || exit 1

BIN="$DIR/subsonic-adapter-go"
LOG_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/subsonic-adapter"
LOG_FILE="$LOG_DIR/adapter.log"
PID_FILE="$LOG_DIR/adapter.pid"

mkdir -p "$LOG_DIR" 2>/dev/null || true

# 桌面通知：有 notify-send 就用，没有就退化成日志。
notify() {
  title=$1
  body=$2
  if command -v notify-send >/dev/null 2>&1; then
    notify-send -a "Subsonic Adapter" -i dialog-error "$title" "$body" >/dev/null 2>&1 || true
  fi
  printf '[launch] %s: %s\n' "$title" "$body" >>"$LOG_FILE" 2>/dev/null || true
}

# 参数解析。
OPEN_BROWSER=1
for arg in "$@"; do
  case "$arg" in
    --no-browser) OPEN_BROWSER=0 ;;
  esac
done

# 前台调试模式。
if [ "${1:-}" = "--foreground" ]; then
  exec "$BIN"
fi

# 开机自启时不弹浏览器：通过环境变量传递给二进制。
if [ "$OPEN_BROWSER" = "0" ]; then
  export NO_BROWSER=1
fi

if [ ! -x "$BIN" ]; then
  notify "启动失败" "找不到可执行文件：$BIN"
  exit 1
fi

# 已在运行则直接打开管理页，避免重复启动抢占端口。
if [ -f "$PID_FILE" ]; then
  OLD_PID=$(cat "$PID_FILE" 2>/dev/null || echo '')
  if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
    if [ "$OPEN_BROWSER" = "1" ]; then
      PORT=$(cat "${XDG_CONFIG_HOME:-$HOME/.config}/subsonic-adapter/port" 2>/dev/null || echo 4535)
      if command -v xdg-open >/dev/null 2>&1; then
        xdg-open "http://127.0.0.1:$PORT" >/dev/null 2>&1 || true
      fi
    fi
    exit 0
  fi
  rm -f "$PID_FILE" 2>/dev/null || true
fi

# 后台启动，完全脱离当前会话：setsid 断开控制终端，nohup 忽略 SIGHUP。
# 这样关掉任何窗口都不会把服务带走。
nohup setsid "$BIN" >>"$LOG_FILE" 2>&1 </dev/null &
NEW_PID=$!
echo "$NEW_PID" >"$PID_FILE" 2>/dev/null || true

# 给进程一点时间尝试监听；若立即退出说明启动失败。
sleep 1.2
if ! kill -0 "$NEW_PID" 2>/dev/null; then
  rm -f "$PID_FILE" 2>/dev/null || true
  TAIL=$(tail -n 5 "$LOG_FILE" 2>/dev/null | tr '\n' ' ')
  notify "Subsonic Adapter 启动失败" "${TAIL:-请查看日志：$LOG_FILE}"
  exit 1
fi

# 启动成功，打开管理页（二进制自身也会尝试打开，这里兜底一次）。
if [ "$OPEN_BROWSER" = "1" ]; then
  PORT=$(cat "${XDG_CONFIG_HOME:-$HOME/.config}/subsonic-adapter/port" 2>/dev/null || echo 4535)
  if command -v xdg-open >/dev/null 2>&1; then
    xdg-open "http://127.0.0.1:$PORT" >/dev/null 2>&1 || true
  fi
fi

exit 0
