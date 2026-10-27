package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// AdapterConfig 运行配置，全部来自环境变量（与 Node 版 config.ts 对齐）。
type AdapterConfig struct {
	Port             int
	Host             string
	UpstreamURL      string
	ServerType       string
	ServerVersion    string
	UpstreamTimeout  time.Duration
	UpstreamCookie   string
}

func intEnv(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func trimURL(v string) string {
	return strings.TrimRight(v, "/")
}

// loadConfig 从环境变量加载配置。
func loadConfig() AdapterConfig {
	host := strings.TrimSpace(os.Getenv("HOST"))
	if host == "" {
		host = "0.0.0.0"
	}
	upstream := strings.TrimSpace(os.Getenv("UPSTREAM_URL"))
	if upstream == "" {
		upstream = "http://127.0.0.1:3200"
	}
	serverType := strings.TrimSpace(os.Getenv("SUBSONIC_TYPE"))
	if serverType == "" {
		serverType = "xiatiao-api"
	}
	serverVersion := strings.TrimSpace(os.Getenv("SUBSONIC_SERVER_VERSION"))
	if serverVersion == "" {
		serverVersion = "0.1.0"
	}
	// 默认端口：优先用上次成功监听的端口，其次 4535。
	// 4533 常被 xiatiao-api 等服务占用，因此不作为默认值。
	defaultPort := loadSavedPort()
	if defaultPort <= 0 {
		defaultPort = 4535
	}
	return AdapterConfig{
		Port:            intEnv("PORT", defaultPort),
		Host:            host,
		UpstreamURL:     trimURL(upstream),
		ServerType:      serverType,
		ServerVersion:   serverVersion,
		UpstreamTimeout: time.Duration(intEnv("UPSTREAM_TIMEOUT_MS", 15000)) * time.Millisecond,
		UpstreamCookie:  os.Getenv("UPSTREAM_COOKIE"),
	}
}
