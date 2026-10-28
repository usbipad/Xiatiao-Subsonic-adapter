# Xiatiao Subsonic API Adapter

本项目是虾条播放器 (Xiatiao Player) 的专属 Subsonic API 兼容网关，主要用于将上游音乐源转换为标准 Subsonic 协议，使虾条播放器及其他兼容 Subsonic 的客户端能够无缝播放。

## 📂 项目结构

```
.
├── subsonic-adapter-go/      # 核心网关服务 (Go 语言实现)
│   ├── packaging/            # 打包脚本与配置 (支持 deb/rpm, amd64/arm64)
│   ├── main.go               # 程序入口
│   └── ...                   # 其他源码文件
├── SUBSONIC_API.md           # 虾条播放器与后端的接口契约文档 (供后端开发参考)
├── .github/workflows/        # GitHub Actions 自动化构建流程
├── .gitignore                # Git 忽略规则
└── README.md                 # 本文件
```

## 🚀 快速开始

核心服务位于 `subsonic-adapter-go` 目录。请进入该目录查看详细的使用和构建说明：

```bash
cd subsonic-adapter-go
cat README.md
```

## 📦 自动化构建

本项目配置了 GitHub Actions，当您推送 `v*` 格式的 Git Tag 时，会自动并行构建 `amd64` 和 `arm64` 架构的 `.deb` 与 `.rpm` 安装包。

## 🙏 致谢

感谢 [qq-music-api](https://github.com/yakult-green-tea/qq-music-api) 提供的音源支持。本项目实现了与其接口的 Subsonic 协议适配，旨在为用户提供更便捷的播放体验。

## 📄 协议

本项目采用 MIT 协议开源。详情请参阅 [LICENSE](LICENSE) 文件。
