#!/bin/sh
# 一键打包：静态编译 → deb → rpm
#
# 用法：
#   ./packaging/build.sh            # 打全部
#   ./packaging/build.sh deb        # 只打 deb
#   ./packaging/build.sh rpm        # 只打 rpm
#   VERSION=1.2.3 ./packaging/build.sh
#
# 产物输出到 ./dist/

set -eu

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
ROOT_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
cd "$ROOT_DIR"

PKG_NAME="subsonic-adapter"
BIN_NAME="subsonic-adapter-go"
VERSION="${VERSION:-0.1.0}"

# 支持通过 TARGET_ARCH 环境变量指定目标架构，默认为 amd64
TARGET_ARCH="${TARGET_ARCH:-amd64}"

case "$TARGET_ARCH" in
  arm64|aarch64)
    GOARCH_TARGET="arm64"
    ARCH_DEB="arm64"
    ARCH_RPM="aarch64"
    ;;
  amd64|x86_64)
    GOARCH_TARGET="amd64"
    ARCH_DEB="amd64"
    ARCH_RPM="x86_64"
    ;;
  *)
    err "不支持的架构: $TARGET_ARCH (仅支持 amd64 或 arm64)"
    ;;
esac

DIST_DIR="$ROOT_DIR/dist"
BUILD_DIR="$DIST_DIR/.build"

GO_BIN="${GO_BIN:-$(command -v go || echo /home/faith/.local/go/bin/go)}"

WHAT="${1:-all}"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
err() { printf '\033[1;31m错误:\033[0m %s\n' "$*" >&2; exit 1; }

[ -x "$GO_BIN" ] || err "找不到 go 工具链（可用 GO_BIN=/path/to/go 指定）"

# ---------- 1. 静态编译 ----------
build_binary() {
  log "静态编译 $BIN_NAME $VERSION"
  rm -rf "$BUILD_DIR"
  mkdir -p "$BUILD_DIR"
  CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH_TARGET" \
    "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$BUILD_DIR/$BIN_NAME" . \
    || err "编译失败"
  chmod 0755 "$BUILD_DIR/$BIN_NAME"
  log "二进制大小: $(du -h "$BUILD_DIR/$BIN_NAME" | cut -f1)"
}

# ---------- 2. 打 deb ----------
build_deb() {
  log "构建 deb"
  DEB_ROOT="$BUILD_DIR/deb-root"
  rm -rf "$DEB_ROOT"
  mkdir -p "$DEB_ROOT/DEBIAN"
  mkdir -p "$DEB_ROOT/usr/bin"
  mkdir -p "$DEB_ROOT/usr/share/applications"
  mkdir -p "$DEB_ROOT/usr/lib/systemd/user"
  mkdir -p "$DEB_ROOT/usr/share/doc/$PKG_NAME"
  mkdir -p "$DEB_ROOT/usr/share/man/man1"
  mkdir -p "$DEB_ROOT/usr/share/$PKG_NAME"

  install -m 0755 "$BUILD_DIR/$BIN_NAME" "$DEB_ROOT/usr/bin/$BIN_NAME"
  # launch.sh 供 XDG autostart 使用（无 systemd 环境下的自启路径）。
  install -m 0755 "$ROOT_DIR/launch.sh" "$DEB_ROOT/usr/share/$PKG_NAME/launch.sh"
  gzip -9n -c "$SCRIPT_DIR/$PKG_NAME.1" >"$DEB_ROOT/usr/share/man/man1/$BIN_NAME.1.gz"
  install -m 0644 "$SCRIPT_DIR/$PKG_NAME.desktop" "$DEB_ROOT/usr/share/applications/$PKG_NAME.desktop"
  install -m 0644 "$SCRIPT_DIR/$PKG_NAME.service" "$DEB_ROOT/usr/lib/systemd/user/$PKG_NAME.service"

  # 图标：按 freedesktop 规范安装到 hicolor 主题的各尺寸目录
  for size in 16 24 32 48 64 128 256 512; do
    ICON_DIR="$DEB_ROOT/usr/share/icons/hicolor/${size}x${size}/apps"
    mkdir -p "$ICON_DIR"
    install -m 0644 "$SCRIPT_DIR/icons/$PKG_NAME-$size.png" "$ICON_DIR/$PKG_NAME.png"
  done
  mkdir -p "$DEB_ROOT/usr/share/icons/hicolor/1024x1024/apps"
  install -m 0644 "$SCRIPT_DIR/icons/$PKG_NAME-1024.png" \
    "$DEB_ROOT/usr/share/icons/hicolor/1024x1024/apps/$PKG_NAME.png"

  cat >"$DEB_ROOT/usr/share/doc/$PKG_NAME/copyright" <<'EOF'
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: subsonic-adapter
Source: https://github.com/usbipad/xiatiao-player

Files: *
Copyright: 2026 Subsonic Adapter contributors
License: MIT
 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:
 .
 The above copyright notice and this permission notice shall be included in all
 copies or substantial portions of the Software.
 .
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 SOFTWARE.
EOF

  # changelog：Debian 要求 native 包必须有
  cat >"$DEB_ROOT/usr/share/doc/$PKG_NAME/changelog" <<EOF
$PKG_NAME ($VERSION) unstable; urgency=medium

  * Release $VERSION.

 -- Subsonic Adapter <noreply@example.com>  $(date -R)
EOF
  gzip -9n "$DEB_ROOT/usr/share/doc/$PKG_NAME/changelog"

  # lintian override：静态链接是刻意的设计（一个包通吃各发行版）
  mkdir -p "$DEB_ROOT/usr/share/lintian/overrides"
  cat >"$DEB_ROOT/usr/share/lintian/overrides/$PKG_NAME" <<EOF
# 刻意静态编译：让同一个包在任意现代 x86_64 Linux 上运行，不依赖 glibc 版本。
$PKG_NAME: statically-linked-binary
EOF

  # control 文件
  SIZE_KB=$(du -sk "$DEB_ROOT" | cut -f1)
  cat >"$DEB_ROOT/DEBIAN/control" <<EOF
Package: $PKG_NAME
Version: $VERSION
Section: sound
Priority: optional
Architecture: $ARCH_DEB
Installed-Size: $SIZE_KB
Maintainer: Subsonic Adapter <noreply@example.com>
Depends: xdg-utils
Recommends: libnotify-bin, gtk-update-icon-cache | gtk2-engines-pixbuf, desktop-file-utils
Description: QQ Music to Subsonic gateway
 A lightweight gateway that exposes a QQ Music API upstream as a
 Subsonic-compatible server, so Subsonic clients can browse and play
 from it. Ships with a local web admin page for login, upstream and
 startup settings.
 .
 The binary is statically linked and has no runtime library deps.
EOF

  # postinst / postrm：刷新图标缓存与桌面数据库，让图标立即生效
  cat >"$DEB_ROOT/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e

# 刷新桌面环境缓存。
#
# 只调 update-desktop-database 往往不够：它依赖 mimeinfo 索引，
# 而 GNOME Shell 把应用列表读进内存，装包后不会自动重扫，
# 表现为「菜单里看不到新应用 / 图标还是旧的」。
# 这里依次刷新索引、图标缓存、XDG 菜单，尽可能让桌面重新识别。
refresh_desktop() {
  if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q /usr/share/applications >/dev/null 2>&1 || true
  fi
  if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
  fi
  if command -v xdg-desktop-menu >/dev/null 2>&1; then
    xdg-desktop-menu forceupdate >/dev/null 2>&1 || true
  fi
}

if [ "$1" = "configure" ]; then
  refresh_desktop
fi

exit 0
EOF

  cat >"$DEB_ROOT/DEBIAN/postrm" <<'EOF'
#!/bin/sh
set -e

if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database /usr/share/applications >/dev/null 2>&1 || true
fi

exit 0
EOF
  chmod 0755 "$DEB_ROOT/DEBIAN/postinst" "$DEB_ROOT/DEBIAN/postrm"

  # 目录权限统一为 0755，避免 lintian 的 non-standard-dir-perm
  find "$DEB_ROOT" -type d -exec chmod 0755 {} +
  find "$DEB_ROOT" -type f -exec chmod 0644 {} +
  chmod 0755 "$DEB_ROOT/usr/bin/$BIN_NAME" "$DEB_ROOT/DEBIAN/postinst" "$DEB_ROOT/DEBIAN/postrm"
  # 脚本必须保留可执行位，否则无法作为 autostart 的 Exec 目标。
  chmod 0755 "$DEB_ROOT/usr/share/$PKG_NAME/launch.sh"

  DEB_FILE="$DIST_DIR/${PKG_NAME}_${VERSION}_${ARCH_DEB}.deb"
  dpkg-deb --root-owner-group --build "$DEB_ROOT" "$DEB_FILE" >/dev/null
  log "已生成: $DEB_FILE"
}

# ---------- 3. 打 rpm ----------
build_rpm() {
  log "构建 rpm"
  command -v rpmbuild >/dev/null 2>&1 || err "未安装 rpmbuild（Fedora: dnf install rpm-build；Debian: apt install rpm）"

  TOP="$BUILD_DIR/rpmbuild"
  rm -rf "$TOP"
  mkdir -p "$TOP"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

  # 以源码包形式喂给 rpmbuild：tar 里含二进制与辅助文件
  STAGE="$BUILD_DIR/stage/$PKG_NAME-$VERSION"
  mkdir -p "$STAGE/icons"
  install -m 0755 "$BUILD_DIR/$BIN_NAME" "$STAGE/$BIN_NAME"
  install -m 0755 "$ROOT_DIR/launch.sh" "$STAGE/launch.sh"
  install -m 0644 "$SCRIPT_DIR/$PKG_NAME.desktop" "$STAGE/$PKG_NAME.desktop"
  install -m 0644 "$SCRIPT_DIR/$PKG_NAME.service" "$STAGE/$PKG_NAME.service"
  for size in 16 24 32 48 64 128 256 512 1024; do
    install -m 0644 "$SCRIPT_DIR/icons/$PKG_NAME-$size.png" "$STAGE/icons/$PKG_NAME-$size.png"
  done
  tar -C "$BUILD_DIR/stage" -czf "$TOP/SOURCES/$PKG_NAME-$VERSION.tar.gz" "$PKG_NAME-$VERSION"

  sed -e "s|@VERSION@|$VERSION|g" \
      -e "s|@BIN_NAME@|$BIN_NAME|g" \
      "$SCRIPT_DIR/$PKG_NAME.spec.in" >"$TOP/SPECS/$PKG_NAME.spec"

  rpmbuild --define "_topdir $TOP" -bb "$TOP/SPECS/$PKG_NAME.spec" >/dev/null

  RPM_FILE=$(find "$TOP/RPMS" -name "*.rpm" | head -n1)
  [ -n "$RPM_FILE" ] || err "rpm 构建产物未找到"
  cp "$RPM_FILE" "$DIST_DIR/"
  log "已生成: $DIST_DIR/$(basename "$RPM_FILE")"
}

mkdir -p "$DIST_DIR"
build_binary

case "$WHAT" in
  all) build_deb; build_rpm ;;
  deb) build_deb ;;
  rpm) build_rpm ;;
  bin) log "仅编译，产物: $BUILD_DIR/$BIN_NAME" ;;
  *) err "未知目标: $WHAT（可选 all|deb|rpm|bin）" ;;
esac

log "完成，产物目录: $DIST_DIR"
ls -lh "$DIST_DIR"/*.deb "$DIST_DIR"/*.rpm 2>/dev/null || true
