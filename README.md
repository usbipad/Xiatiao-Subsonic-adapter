# Xiatiao Subsonic Adapter

一个轻量级的网关服务，将上游音乐 API（如 QQ 音乐）转换为标准 Subsonic 协议。专为虾条播放器设计，同时也兼容任何支持 Subsonic API 的客户端。

## ✨ 主要特性

*   **协议兼容**：完整实现 Subsonic API 1.16.1 标准。
*   **多架构支持**：原生支持 `amd64` 与 `arm64` (aarch64) 架构。
*   **静态编译**：无外部 C 依赖，单二进制文件即可运行。
*   **便捷打包**：提供脚本一键生成 `.deb` 与 `.rpm` 安装包。

## 🚀 快速开始

### 1. 获取源码
```bash
git clone https://github.com/usbipad/Xiatiao-Subsonic-adapter.git
cd Xiatiao-Subsonic-adapter/subsonic-adapter-go
```

### 2. 本地编译
确保已安装 **Go 1.21+**，然后执行：
```bash
go build -o subsonic-adapter-go .
./subsonic-adapter-go
```

### 3. 打包安装包 (可选)
如果您需要生成系统安装包，可以使用内置脚本：
```bash
# 构建 amd64 版本
./packaging/build.sh all

# 交叉编译 arm64 版本
TARGET_ARCH=arm64 VERSION=1.0.0 ./packaging/build.sh all
```
产物将生成在 `dist/` 目录下。

推送到 `v*` tag 时，GitHub Actions 会自动为 `amd64`/`arm64` 构建安装包，并把 deb、rpm 与 `SHA256SUMS.txt` 发布到对应的 Release：
```bash
git tag v1.0.0
git push origin v1.0.0
```

> 更多详细的使用说明、启动方式及配置指南，请参阅 [subsonic-adapter-go/README.md](subsonic-adapter-go/README.md)。

## 💻 安装与使用

如果您已经通过 `.deb` 或 `.rpm` 包安装了本程序，可以通过以下方式使用：

### 1. 命令行直接运行
```bash
subsonic-adapter-go
```
启动后，程序通常会自动在浏览器中打开本地 Web 管理页面。按 `Ctrl + C` 即可停止运行。

### 2. 作为后台服务运行（🌟 推荐）
安装包已内置 systemd 用户服务，适合在后台稳定运行：
```bash
# 启动服务
systemctl --user start subsonic-adapter

# 设置开机/登录自动启动
systemctl --user enable subsonic-adapter

# 查看运行状态与实时日志
journalctl --user -u subsonic-adapter -f
```

### 3. 桌面环境启动
在 Linux 桌面环境的应用菜单中搜索 **Subsonic Adapter**，点击图标即可启动并自动打开管理页面。

> **💡 提示**：启动服务后，请先在 Web 管理页面配置上游 API 地址并完成授权，随后即可在虾条播放器或其他 Subsonic 客户端（如 Navidrome, Substreamer 等）中添加服务器地址进行播放。

## 📚 相关文档

*   **[SUBSONIC_API.md](SUBSONIC_API.md)**: 虾条播放器与后端的接口契约，供开发者参考。
*   **[packaging/README.md](subsonic-adapter-go/packaging/README.md)**: 关于打包脚本的详细参数说明。

## 🙏 致谢

感谢 [qq-music-api](https://github.com/yakult-green-tea/qq-music-api) 提供的音源支持。本项目实现了与其接口的 Subsonic 协议适配，旨在为用户提供更便捷的播放体验。

## 📄 协议

本项目采用 MIT 协议开源。详情请参阅 [LICENSE](LICENSE) 文件。
