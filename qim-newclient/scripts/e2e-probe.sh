#!/usr/bin/env bash
# 真服务端端到端联调：串行跑全部协议探针，覆盖
#   登录 → 会话 → 消息 → 组织架构 → 群 / 频道 / 通知 / 文件 / 资料 等链路。
#
# 用法：
#   QIM_PASS='密码' ./scripts/e2e-probe.sh [服务端地址] [用户名]
#
# 默认 -server http://127.0.0.1:8080 -user native_probe；
# 密码只从环境变量 QIM_PASS 读，不写进命令行（避免进 shell 历史）。
#
# 注意：探针会向服务端写入并清理测试数据（临时建群、发消息、置顶翻转等），
#       仅用于本地开发库，不要指向生产环境。

set -u

SERVER="${1:-http://127.0.0.1:8080}"
USER="${2:-native_probe}"
BIN="${QIM_E2E_BIN:-/tmp/qim-e2e-probe}"
LOGDIR="${QIM_E2E_LOGDIR:-/tmp/qim-e2e-logs}"

if [ -z "${QIM_PASS:-}" ]; then
	echo "请设置 QIM_PASS 环境变量后再跑（密码不进命令行历史）" >&2
	exit 2
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT" || exit 2

echo "编译探针二进制 ……"
go build -o "$BIN" . || exit 2
mkdir -p "$LOGDIR"

PROBES='probe-api probe-ai probe-msg probe-convlist probe-c probe-groups probe-actions probe-media probe-profile probe-quick probe-files probe-channels probe-recall probe-readusers probe-announce probe-notifs'

pass=0
fail=0
echo "服务端: $SERVER  账号: $USER"
echo "----------------------------------------"
for p in $PROBES; do
	if QIM_PASS="$QIM_PASS" "$BIN" "-$p" -server "$SERVER" -user "$USER" >"$LOGDIR/$p.log" 2>&1; then
		printf '  ✓ %-16s\n' "$p"
		pass=$((pass + 1))
	else
		rc=$?
		printf '  ✗ %-16s exit=%d  （日志: %s/%s.log）\n' "$p" "$rc" "$LOGDIR" "$p"
		fail=$((fail + 1))
	fi
done

echo "----------------------------------------"
echo "结果: $pass 通过 / $fail 失败"
[ "$fail" -eq 0 ]
