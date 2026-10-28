# Changelog

All notable changes to this project will be documented in this file.

## [v1.0.1] - 2026-10-28

### 🐛 修复

*   **播放偶发卡死**：音频代理此前每次请求都新建 `http.Client` + `Transport`，连接池每次都是空的，每个 Range 请求都要重新与上游 CDN 建 TCP + TLS。实测 TTFB 呈 0.2 s ~ 15.7 s 双峰分布，且播放若干次后会彻底静默（TCP 能连上但一个字节都不返回，播放器表现为「状态显示播放中但没有声音」，只能手动切歌）。
*   改为进程级复用 `streamClient`：`MaxIdleConnsPerHost=8` / `MaxConnsPerHost=16`，`IdleConnTimeout=90s`，不设总超时（避免长曲目被截断），靠 Dial / ResponseHeader 超时与 ctx 取消兜底；不开 HTTP/2（Range 在 HTTP/1.1 下每连接一请求，天然错开更稳）。

### 📝 文档

*   README 改为精简结构，构建与使用说明统一指向 `subsonic-adapter-go/README.md`。

---

## [v1.0.0] - 2026-10-27

### 🎉 初始发布

这是 Xiatiao Subsonic Adapter 的第一个正式版本。本项目旨在将上游音源转换为标准 Subsonic 协议，适配虾条播放器。

### ✨ 主要特性

*   **协议兼容**：完整实现 Subsonic API 1.16.1，支持搜索、播放、歌词及歌单管理。
*   **多架构支持**：原生支持 `amd64` (x86_64) 与 `arm64` (aarch64) 架构。
*   **便捷安装**：提供 `.deb` (Debian/Ubuntu) 和 `.rpm` (Fedora/RHEL) 安装包。
*   **静态编译**：无外部依赖，单二进制文件即可运行，资源占用极低。
*   **Web 管理页**：内置本地 Web 界面，方便配置上游地址与登录状态。
*   **播放更顺**：音频代理复用进程级 HTTP 连接，seek 与断流重连不再
    每次重新与上游建连（此前 TTFB 呈 0.2s~15.7s 双峰分布，用户会听到数秒静音）。

### 📦 安装包说明

*   `subsonic-adapter_*_amd64.deb/rpm`: 适用于 Intel/AMD 处理器。
*   `subsonic-adapter_*_arm64.deb/rpm`: 适用于 ARM 处理器（如树莓派、Apple Silicon 虚拟机等）。

### 🙏 致谢

感谢 [qq-music-api](https://github.com/yakult-green-tea/qq-music-api) 提供的音源支持。
