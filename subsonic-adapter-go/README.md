# Subsonic Adapter (Go)

一个轻量级的网关服务，将上游音乐 API（如 QQ 音乐）暴露为兼容 Subsonic 协议的服务器。支持本地 Web 管理页面，用于配置登录、上游地址和启动设置。

## ✨ 特性

- **标准兼容**：完全兼容 Subsonic API 1.16.1，支持虾条播放器及 Navidrome 等主流客户端。
- **静态编译**：使用 `CGO_ENABLED=0` 编译，无外部 C 依赖，一个二进制文件即可运行。
- **多架构支持**：原生支持 `amd64` 和 `arm64` (aarch64) 架构的打包与运行。
- **开箱即用**：提供 `.deb` 和 `.rpm` 安装包，支持桌面图标启动、开机自启及 systemd 用户服务。

## 🚀 快速开始

### 1. 本地编译运行

确保已安装 Go 1.21+：

```bash
go build -o subsonic-adapter-go .
./subsonic-adapter-go
```

### 2. 使用安装包 (推荐)

见下方「打包与安装」章节。

## 📦 打包与安装

项目提供了自动化打包脚本，支持生成 `deb` 和 `rpm` 格式的安装包。

### 构建命令

```bash
./packaging/build.sh          # 同时构建 deb + rpm
./packaging/build.sh deb      # 仅构建 deb
./packaging/build.sh rpm      # 仅构建 rpm
./packaging/build.sh bin      # 仅编译二进制文件
VERSION=1.2.3 ./packaging/build.sh   # 指定版本号
```
产物将输出到 `dist/` 目录。

**构建依赖**：
- Go 1.21+ (可通过 `GO_BIN=/path/to/go` 指定路径)
- `dpkg-deb` (用于构建 deb)
- `rpmbuild` (用于构建 rpm，Debian/Ubuntu 下可通过 `apt install rpm` 安装)

### 安装与使用

**Debian/Ubuntu:**
```bash
sudo dpkg -i dist/subsonic-adapter_*_amd64.deb   # 或 arm64.deb
```

**Fedora/RHEL/CentOS:**
```bash
sudo rpm -i dist/subsonic-adapter-*.x86_64.rpm   # 或 aarch64.rpm
```

安装后，您可以通过以下三种方式之一启动服务：
1. **桌面图标**：在应用菜单中搜索「Subsonic 适配器」点击启动（会自动打开管理页）。
2. **开机自启**：打开管理页 →「启动设置」→ 打开开关（写入 `~/.config/autostart/`）。
3. **systemd 用户服务**（最可靠，不依赖桌面会话）：
   ```bash
   systemctl --user daemon-reload
   systemctl --user enable --now subsonic-adapter
   ```

### 卸载

```bash
sudo dpkg -r subsonic-adapter   # Debian/Ubuntu
sudo rpm -e subsonic-adapter    # Fedora/RHEL
```
*(注：用户配置目录 `~/.config/subsonic-adapter/` 和 `~/.local/state/subsonic-adapter/` 不会被自动删除，如需彻底清理请手动移除。)*

## 📁 包内文件结构

| 路径 | 说明 |
|---|---|
| `/usr/bin/subsonic-adapter-go` | 静态编译的二进制可执行文件 |
| `/usr/share/applications/subsonic-adapter.desktop` | 桌面应用菜单入口 |
| `/usr/share/icons/hicolor/*/apps/subsonic-adapter.png` | 应用图标 (支持 16 到 1024 多种尺寸) |
| `/usr/lib/systemd/user/subsonic-adapter.service` | systemd 用户服务单元 (默认不启用) |

## ⚙️ CI/CD 自动化

本项目配置了 GitHub Actions (`.github/workflows/release.yml`)。当推送 `v*` 格式的 Git Tag 时，会自动触发并行构建，生成 `amd64` 和 `arm64` 架构的安装包，并作为 Artifacts 上传。

## 🙏 致谢

感谢 [qq-music-api](https://github.com/yakult-green-tea/qq-music-api) 提供的音源支持。本项目实现了与其接口的 Subsonic 协议适配，旨在为用户提供更便捷的播放体验。

## 📄 协议

本项目采用 MIT 协议开源。详情请参阅根目录的 [LICENSE](../LICENSE) 文件。
