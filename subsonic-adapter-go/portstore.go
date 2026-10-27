package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 端口记忆：把上次成功监听的端口写到用户配置目录，下次启动优先复用，
// 避免因自动顺延导致每次启动端口都变化。

// portStatePath 返回端口记录的存放路径。
// 优先 XDG_CONFIG_HOME，其次 ~/.config；无法确定时返回空串表示不持久化。
func portStatePath() string {
	dir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "subsonic-adapter", "port")
}

// loadSavedPort 读取上次成功监听的端口；没有记录或内容非法时返回 0。
func loadSavedPort() int {
	path := portStatePath()
	if path == "" {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n <= 0 || n > 65535 {
		return 0
	}
	return n
}

// loadSavedPortOrDefault 返回记录端口，没有记录时回退到默认 4535。
func loadSavedPortOrDefault() int {
	if p := loadSavedPort(); p > 0 {
		return p
	}
	return 4535
}

// savePort 记录本次成功监听的端口。写入失败只记日志，不影响服务。
func savePort(port int) {
	path := portStatePath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("无法创建端口记录目录: %v", err)
		return
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(port)), 0o644); err != nil {
		log.Printf("无法写入端口记录: %v", err)
	}
}
