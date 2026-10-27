# Changelog

All notable changes to this project will be documented in this file.

## [v1.0.0] - 2026-10-27

### 🎉 初始发布

这是 Xiatiao Subsonic Adapter 的第一个正式版本。本项目旨在将上游音源转换为标准 Subsonic 协议，适配虾条播放器。

### ✨ 主要特性

*   **协议兼容**：完整实现 Subsonic API 1.16.1，支持搜索、播放、歌词及歌单管理。
*   **多架构支持**：原生支持 `amd64` (x86_64) 与 `arm64` (aarch64) 架构。
*   **便捷安装**：提供 `.deb` (Debian/Ubuntu) 和 `.rpm` (Fedora/RHEL) 安装包。
*   **静态编译**：无外部依赖，单二进制文件即可运行，资源占用极低。
*   **Web 管理页**：内置本地 Web 界面，方便配置上游地址与登录状态。

### 📦 安装包说明

*   `subsonic-adapter_*_amd64.deb/rpm`: 适用于 Intel/AMD 处理器。
*   `subsonic-adapter_*_arm64.deb/rpm`: 适用于 ARM 处理器（如树莓派、Apple Silicon 虚拟机等）。

### 🙏 致谢

感谢 [qq-music-api](https://github.com/yakult-green-tea/qq-music-api) 提供的音源支持。
