#!/bin/sh
# after-install-linux.sh — deb 安装时为所有已有用户在桌面放置 QIM 快捷方式
# 由 electron-builder 嵌入 postinst，以 root 身份执行。
# 任何失败都只记录、不阻断安装（最终 exit 0）。
#
# 测试时可用 QIM_MAIN=0 source 本文件，仅加载函数、不执行主流程。

: "${QIM_MAIN:=1}"

# electron-builder 把本脚本当模板：executable 变量在打包时替换为 linux.executableName 的值。
# .desktop 文件名 = <executableName>.desktop（FpmTarget 安装到 /usr/share/applications/）。
SYSTEM_DESKTOP="/usr/share/applications/${executable}.desktop"

# resolve_desktop_dir 解析用户桌面目录：优先 XDG_DESKTOP_DIR，回退 Desktop / 桌面
# 参数：$1 = 用户家目录。输出桌面目录绝对路径（无则不输出）。
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

# install_for_user 为单个用户复制快捷方式到其桌面
install_for_user() {
  user="$1"
  home="$2"
  desktop_dir=$(resolve_desktop_dir "$home")
  [ -z "$desktop_dir" ] && return
  target="$desktop_dir/qim.desktop"
  if cp "$SYSTEM_DESKTOP" "$target" 2>/dev/null; then
    chmod 755 "$target" 2>/dev/null || true
    chown "$user" "$target" 2>/dev/null || true
    echo "QIM: 已为 $user 创建桌面快捷方式 $target"
  fi
}

if [ "$QIM_MAIN" = "1" ]; then
  if [ ! -f "$SYSTEM_DESKTOP" ]; then
    echo "QIM: 系统菜单项 $SYSTEM_DESKTOP 不存在，跳过桌面快捷方式创建"
    exit 0
  fi

  # 优先 getent（含 LDAP/SSSD 用户），回退 /etc/passwd
  if command -v getent >/dev/null 2>&1; then
    passwd_src=$(getent passwd)
  else
    passwd_src=$(cat /etc/passwd)
  fi

  # 遍历 UID >= 1000 的普通用户
  echo "$passwd_src" | while IFS=: read -r name _ uid _ _ home _; do
    case "$uid" in
      ''|*[!0-9]*) continue ;;
    esac
    if [ "$uid" -ge 1000 ] 2>/dev/null && [ "$name" != "nobody" ] && [ -d "$home" ]; then
      install_for_user "$name" "$home"
    fi
  done

  exit 0
fi
