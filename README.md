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

> 更多详细的使用说明、启动方式及配置指南，请参阅 [subsonic-adapter-go/README.md](subsonic-adapter-go/README.md)。

## 📚 相关文档

*   **[SUBSONIC_API.md](SUBSONIC_API.md)**: 虾条播放器与后端的接口契约，供开发者参考。
*   **[packaging/README.md](subsonic-adapter-go/packaging/README.md)**: 关于打包脚本的详细参数说明。

## 🙏 致谢

感谢 [qq-music-api](https://github.com/yakult-green-tea/qq-music-api) 提供的音源支持。本项目实现了与其接口的 Subsonic 协议适配，旨在为用户提供更便捷的播放体验。

## 📄 协议

本项目采用 MIT 协议开源。详情请参阅 [LICENSE](LICENSE) 文件。
