# 打包说明

## 构建

    ./packaging/build.sh          # 打 deb + rpm（默认 amd64）
    ./packaging/build.sh deb      # 只打 deb
    ./packaging/build.sh rpm      # 只打 rpm
    ./packaging/build.sh bin      # 只编译二进制
    VERSION=1.2.3 ./packaging/build.sh   # 指定版本号
    TARGET_ARCH=arm64 ./packaging/build.sh   # 交叉编译 arm64 架构

产物输出到 `dist/`。

**多架构支持**：通过 `TARGET_ARCH` 环境变量指定目标架构：
- `amd64`（默认）：x86_64 架构，deb 包标记为 `amd64`，rpm 包标记为 `x86_64`
- `arm64`：ARM64 架构，deb 包标记为 `arm64`，rpm 包标记为 `aarch64`

依赖：

- 编译：Go 1.21+（可用 `GO_BIN=/path/to/go` 指定）
- deb：`dpkg-deb`
- rpm：`rpmbuild`（Fedora 用 dnf install rpm-build；Debian 用 apt install rpm）

## 关于「一个包通吃」

二进制以 `CGO_ENABLED=0` 静态编译，`ldd` 显示无动态依赖，因此同一个 deb/rpm 可以在任何现代 Linux 上运行，不受 glibc 版本影响。

**交叉编译**：Go 的交叉编译能力使得在 x86_64 机器上可以直接编译出 arm64 二进制，无需额外的交叉编译工具链。GitHub Actions 会自动并行构建两种架构的安装包。

## 安装

    sudo dpkg -i dist/subsonic-adapter_0.1.0_amd64.deb
    sudo rpm -i dist/subsonic-adapter-0.1.0-1.x86_64.rpm

装上后：

- 命令行：`subsonic-adapter-go`
- 应用菜单：**Subsonic 适配器**（会自动打开管理页）

## 安装后启用方式（三选一）

**1. 桌面图标启动**（推荐日常使用）

应用菜单点「Subsonic 适配器」，后台运行并打开管理页。

**2. 开机自启**

打开管理页 → 「启动设置」→ 打开开关。写入 `~/.config/autostart/`，登录桌面后自动后台启动。

**3. systemd 用户服务**（最可靠，不依赖桌面会话）

    systemctl --user daemon-reload
    systemctl --user enable --now subsonic-adapter

该单元已设 `NO_BROWSER=1`，启动时不弹浏览器。

## 卸载

    sudo dpkg -r subsonic-adapter
    sudo rpm -e subsonic-adapter

用户配置（`~/.config/subsonic-adapter/`、`~/.local/state/subsonic-adapter/`）不会被删除，如需彻底清理请手动删除。

## 包内文件

| 路径 | 说明 |
|---|---|
| `/usr/bin/subsonic-adapter-go` | 静态二进制 |
| `/usr/share/applications/subsonic-adapter.desktop` | 应用菜单入口 |
| `/usr/share/icons/hicolor/*/apps/subsonic-adapter.png` | 应用图标（9 种尺寸） |
| `/usr/lib/systemd/user/subsonic-adapter.service` | systemd 用户服务（默认不启用） |

## 图标

图标源图为 `packaging/icons/subsonic-adapter-1024.png`，构建时不做缩放，直接使用预先从源图生成的 16/24/32/48/64/128/256/512/1024 九种尺寸，安装到 freedesktop 规定的 `hicolor` 主题目录。

如需更换图标，替换 1024 源图后重新生成各尺寸：

    python3 -c "from PIL import Image; s=Image.open('packaging/icons/subsonic-adapter-1024.png').convert('RGBA'); [s.resize((n,n), Image.LANCZOS).save(f'packaging/icons/subsonic-adapter-{n}.png', optimize=True) for n in (16,24,32,48,64,128,256,512)]"

安装后包管理器会刷新图标缓存与桌面数据库（deb 的 postinst、rpm 的 %post），图标无需注销即可生效。

## 注意

- 上游 QQ 音乐 API 服务**不包含在此包内**，需自行部署运行，然后在管理页填写上游地址。
- 包内不自动启用任何自启机制，由用户自行决定。
