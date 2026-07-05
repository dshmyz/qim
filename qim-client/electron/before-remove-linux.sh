#!/bin/sh
# before-remove-linux.sh — deb 卸载时清理各用户桌面上的 QIM 快捷方式
# 由 electron-builder 嵌入 prerm，以 root 身份执行。
#
# 测试时可用 QIM_MAIN=0 source 本文件，仅加载函数、不执行主流程。

: "${QIM_MAIN:=1}"

# electron-builder 把本脚本当模板：executable 变量在打包时替换为 linux.executableName 的值。
DESKTOP_FILENAME="${executable}.desktop"

# resolve_desktop_dir 与 after-install-linux.sh 保持一致（两脚本各自嵌入，无法共享）
resolve_desktop_dir() {
  home="$1"
  dirs_file="$home/.config/user-dirs.dirs"
  if [ -f "$dirs_file" ]; then
    xdg=$(grep '^XDG_DESKTOP_DIR=' "$dirs_file" 2>/dev/null | head -1 | \
      sed -e 's/XDG_DESKTOP_DIR=//' -e 's/^"//' -e 's/"$//' -e "s|\$HOME|$home|")
    if [ -n "$xdg" ] && [ -d "$xdg" ]; then
      echo "$xdg"
      return
    fi
  fi
  for name in Desktop 桌面; do
    if [ -d "$home/$name" ]; then
      echo "$home/$name"
      return
    fi
  done
}

if [ "$QIM_MAIN" = "1" ]; then
  if command -v getent >/dev/null 2>&1; then
    passwd_src=$(getent passwd)
  else
    passwd_src=$(cat /etc/passwd)
  fi

  echo "$passwd_src" | while IFS=: read -r name _ uid _ _ home _; do
    case "$uid" in
      ''|*[!0-9]*) continue ;;
    esac
    if [ "$uid" -ge 1000 ] 2>/dev/null && [ -d "$home" ]; then
      desktop_dir=$(resolve_desktop_dir "$home")
      if [ -n "$desktop_dir" ] && [ -f "$desktop_dir/$DESKTOP_FILENAME" ]; then
        rm -f "$desktop_dir/$DESKTOP_FILENAME" 2>/dev/null && echo "QIM: 已移除 $name 的桌面快捷方式"
      fi
    fi
  done

  exit 0
fi
