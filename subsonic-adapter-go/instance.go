package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// 单实例互斥。
//
// 背景：程序可能被多种方式启动（桌面图标、systemd、命令行、autostart），
// 若各自独立判断端口，会出现多实例并占用多个端口。
// 这里用一个全局 PID 文件做互斥：任何启动方式都先检查它。

// instancePIDPath 返回 PID 文件路径，放在用户状态目录下，所有启动方式共享。
func instancePIDPath() string {
	dir := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "subsonic-adapter", "instance.pid")
}

// readPIDFile 读取 PID 文件；无效或不存在返回 0。
func readPIDFile(path string) int {
	if path == "" {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// processAlive 判断 PID 对应的进程是否仍存活。
// 用 signal 0 探测：不发送信号，只做存在性检查。
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// acquireInstanceLock 尝试取得单实例锁。
// 成功返回自身 PID，调用方负责在退出时释放；
// 若已有存活实例，返回该实例的 PID 与 false。
func acquireInstanceLock() (existingPID int, acquired bool) {
	path := instancePIDPath()
	if path == "" {
		// 无法确定状态目录时不阻塞启动，退化为无锁运行。
		return 0, true
	}

	if pid := readPIDFile(path); pid > 0 && processAlive(pid) && pid != os.Getpid() {
		return pid, false
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("无法创建状态目录: %v（继续启动，不启用单实例保护）", err)
		return 0, true
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		log.Printf("无法写入 PID 文件: %v（继续启动，不启用单实例保护）", err)
		return 0, true
	}
	return 0, true
}

// releaseInstanceLock 删除 PID 文件（仅当内容仍是自己的 PID）。
func releaseInstanceLock() {
	path := instancePIDPath()
	if path == "" {
		return
	}
	if readPIDFile(path) != os.Getpid() {
		return
	}
	_ = os.Remove(path)
}

// instanceAlreadyRunningMessage 生成「已有实例在运行」的提示。
func instanceAlreadyRunningMessage(pid int) string {
	port := loadSavedPort()
	if port <= 0 {
		port = 4535
	}
	return fmt.Sprintf("已有实例在运行（pid=%d），管理页面：http://127.0.0.1:%d", pid, port)
}
