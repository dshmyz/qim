#!/usr/bin/env bash
#
# MCP/CLI Server 一键安装脚本
#
# 用法：
#   SERVER=http://your-server bash -c "$(curl -fsSL http://your-server/install.sh)"
#   或
#   curl -fsSL http://your-server/install.sh | SERVER=http://your-server bash
#
# 环境变量：
#   SERVER      服务器地址，默认 http://localhost:8080
#   PRODUCT     产物类型，默认 mcp（mcp | cli）
#   INSTALL_DIR  安装目录，默认 $HOME/.local/bin
#   BIN_NAME     安装后的文件名（默认取服务器返回的上传原名）
#
# 行为：检测平台 → 查询最新版本与 SHA256 → 下载预编译二进制（文件名用上传时的原名）
#       → SHA256 校验 → 安装到 PATH → 打印配置指引。
set -euo pipefail

SERVER="${SERVER:-http://localhost:8080}"
PRODUCT="${PRODUCT:-mcp}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
OVERRIDE_BIN_NAME="${BIN_NAME:-}"

say() { printf '\033[1;32m%s\033[0m\n' "$*"; }
err() { printf '\033[1;31m错误: %s\033[0m\n' "$*" >&2; }

# ---------- 1. 检测平台 ----------
case "$(uname -s)" in
  Darwin) OS=darwin ;;
  Linux)  OS=linux  ;;
  MINGW*|MSYS*|CYGWIN*) OS=windows ;;
  *) err "不支持的 OS: $(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) err "不支持的架构: $(uname -m)"; exit 1 ;;
esac
echo "检测平台: $OS/$ARCH (服务器: $SERVER, 产物: $PRODUCT)"

# ---------- 2. 查询版本 ----------
# 单个 os/arch + product 的查询只返回该平台最新版本，sha256 唯一的 key 即上传时的文件名。
VERSION_URL="$SERVER/api/v1/cli/version?os=$OS&arch=$ARCH&product=$PRODUCT"
if ! VERSION_JSON="$(curl -fsSL --connect-timeout 10 "$VERSION_URL" 2>/dev/null)"; then
  err "无法访问服务器: $VERSION_URL"
  err "请检查 SERVER 地址是否正确、服务器是否在线。"
  exit 1
fi

# 优先用 python3 解析 JSON（macOS 自带、绝大多数 Linux 发行版也自带），
# 避免 sed 解析在 JSON 字段顺序变化 / sha256 编码方式调整时静默失败。
# python3 不可用时回退到 sed 提取（仅作为兜底）。
parse_json() {
  # $1 = 字段名：version | sha256_key | sha256_val
  # 兼容两种返回结构：直接 {version,sha256} 或 {code,data:{version,sha256}}
  # 用 -c 把代码作为参数传入，stdin 留给 JSON 数据。
  python3 -c '
import json, sys
field = sys.argv[1]
raw = sys.stdin.read()
data = json.loads(raw)
payload = data.get("data") if isinstance(data.get("data"), dict) else data
if field == "version":
    print(payload.get("version") or "")
elif field in ("sha256_key", "sha256_val"):
    sha = payload.get("sha256") or {}
    if not sha:
        print("")
    elif field == "sha256_key":
        print(next(iter(sha.keys())))
    else:
        print(next(iter(sha.values())))
' "$1" 2>/dev/null || return 1
}

fallback_sed_version() {
  printf '%s' "$VERSION_JSON" | sed -n 's/.*"version":"\([^"]*\)".*/\1/p'
}

fallback_sed_sha256_key() {
  printf '%s' "$VERSION_JSON" | tr -d '\n' | sed -n 's/.*"sha256": *{\s*"\([^"]*\)":"\([0-9a-fA-F]\{64\}\)".*/\1/p'
}

fallback_sed_sha256_val() {
  printf '%s' "$VERSION_JSON" | tr -d '\n' | sed -n 's/.*"sha256": *{\s*"[^"]*":"\([0-9a-fA-F]\{64\}\)".*/\1/p'
}

if command -v python3 >/dev/null 2>&1; then
  VERSION="$(printf '%s' "$VERSION_JSON" | parse_json version || true)"
  BIN_NAME="$(printf '%s' "$VERSION_JSON" | parse_json sha256_key || true)"
  EXPECTED="$(printf '%s' "$VERSION_JSON" | parse_json sha256_val || true)"
else
  # 无 python3：用 sed 兜底，但提示用户解析可能不稳。
  echo "警告: 未找到 python3，使用 sed 解析 JSON（对字段顺序敏感，可能不稳）。"
  VERSION="$(fallback_sed_version)"
  BIN_NAME="$(fallback_sed_sha256_key)"
  EXPECTED="$(fallback_sed_sha256_val)"
fi

if [ -z "$VERSION" ] || [ "$VERSION" = "unknown" ]; then
  err "服务器尚未发布 $PRODUCT 的 $OS/$ARCH 版本（可能未在管理后台发布，或服务器地址不对）"
  err "接口返回: $VERSION_JSON"
  exit 1
fi

if [ -z "$BIN_NAME" ] || [ -z "$EXPECTED" ]; then
  err "服务器未返回有效的二进制名/校验值（该版本可能未配置 SHA256）"
  err "接口返回: $VERSION_JSON"
  exit 1
fi
[ -n "$OVERRIDE_BIN_NAME" ] && BIN_NAME="$OVERRIDE_BIN_NAME"
echo "最新版本: $VERSION (二进制: $BIN_NAME)"

# ---------- 3. 下载二进制 ----------
DEST="$INSTALL_DIR/$BIN_NAME"
TMP="$(mktemp "${TMPDIR:-/tmp}/install.XXXXXX")"
trap 'rm -f "$TMP"' EXIT

DOWNLOAD_URL="$SERVER/api/v1/cli/download?os=$OS&arch=$ARCH&product=$PRODUCT"
echo "下载 → $TMP"
if ! curl -fsSL --connect-timeout 30 "$DOWNLOAD_URL" -o "$TMP"; then
  err "下载失败: $DOWNLOAD_URL"
  err "请确认服务器已发布 $PRODUCT/$OS/$ARCH 的二进制。"
  exit 1
fi

# ---------- 4. SHA256 校验（fail-closed） ----------
# macOS 自带 shasum；Linux 一般有 sha256sum；都没有则报错。
if command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "$TMP" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$TMP" | awk '{print $1}')"
else
  err "系统缺少 shasum 与 sha256sum，无法校验完整性。请安装 coreutils 或 perl 后重试。"
  exit 1
fi
if [ "$ACTUAL" != "$EXPECTED" ]; then
  err "SHA256 校验失败（期望 $EXPECTED，实际 $ACTUAL）"
  exit 1
fi
echo "SHA256 校验通过"

# ---------- 5. 安装 ----------
mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP" "$DEST"
rm -f "$TMP"
trap - EXIT
say "已安装: $DEST (v$VERSION)"

# ---------- 6. 确保 PATH ----------
if ! printf ':%s:' "$PATH" | grep -q ":$INSTALL_DIR:"; then
  case "$SHELL" in
    *zsh) RC="$HOME/.zshrc" ;;
    *bash) RC="$HOME/.bashrc" ;;
    *) RC="$HOME/.profile" ;;
  esac
  if ! grep -qF "export PATH=\"$INSTALL_DIR" "$RC" 2>/dev/null; then
    printf '\n# 已安装 CLI 产物\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$RC"
    echo "已将 $INSTALL_DIR 加入 $RC（新终端生效，或执行: source $RC）"
  fi
fi

cat <<EOF

✅ 安装完成: $BIN_NAME (v$VERSION)

接下来在 Claude Code (~/.claude/settings.json) 里配置:
{
  "mcpServers": {
    "nuim": {
      "command": "$BIN_NAME",
      "args": ["--token", "qbot_your_token_here", "--server", "$SERVER"]
    }
  }
}
（Cursor 请在 .cursor/mcp.json 中做相同配置）
EOF
