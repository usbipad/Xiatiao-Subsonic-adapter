#!/bin/sh
# subsonic-adapter-go 启动脚本
# 用法：
#   ./start.sh
#   UPSTREAM_URL=http://127.0.0.1:3200 PORT=4535 ./start.sh
set -e
cd "$(dirname "$0")"

if [ ! -x ./subsonic-adapter-go ]; then
  echo "错误：未找到 ./subsonic-adapter-go，请先构建：" >&2
  echo "  /home/faith/.local/go/bin/go build -o subsonic-adapter-go ." >&2
  exit 1
fi

export UPSTREAM_URL="${UPSTREAM_URL:-http://127.0.0.1:3200}"

# 单实例保护由二进制自身负责：它会检查全局 PID 文件，
# 若已有实例在运行则打印提示并退出，不会抢占第二个端口。
# 这里不再硬编码 PORT，让二进制按「历史记录 → 4535」的顺序决定，
# 并保留显式 PORT 的最高优先级。

echo "subsonic-adapter-go 启动中..."
echo "  上游: $UPSTREAM_URL"
[ -n "${PORT:-}" ] && echo "  端口: $PORT（显式指定）"

exec ./subsonic-adapter-go
