#!/bin/sh

set -e

SUDOERS_FILE="/etc/sudoers.d/qim-update"
SUDOERS_SOURCE="/opt/qim/resources/qim-update.sudoers"

if [ -f "$SUDOERS_SOURCE" ]; then
  cp "$SUDOERS_SOURCE" "$SUDOERS_FILE"
  chmod 440 "$SUDOERS_FILE"
  echo "QIM: sudoers rule installed to $SUDOERS_FILE"
else
  echo "QIM: sudoers source not found at $SUDOERS_SOURCE, creating inline"
  echo "# QIM 自动更新免密 sudo 规则" > "$SUDOERS_FILE"
  echo "# 允许 sudo 组的用户无需密码运行更新安装脚本" >> "$SUDOERS_FILE"
  echo "%sudo ALL=(ALL) NOPASSWD: /opt/qim/resources/install-update-linux.sh *" >> "$SUDOERS_FILE"
  chmod 440 "$SUDOERS_FILE"
fi
