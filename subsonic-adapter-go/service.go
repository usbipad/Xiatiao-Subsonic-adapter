package main

import (
	"log"
	"net/http"
	"os"
	"syscall"
	"time"
)

// 服务自身管理。注意边界：本进程只能管理“自己”，上游由用户自行部署运行。

// restartSelf 用当前可执行文件与相同参数原地重启进程。
// 走 syscall.Exec，pid 不变，监听端口在极短时间内释放再占用。
func restartSelf() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	log.Printf("正在重启服务: %s %v", exePath, os.Args[1:])
	// 给 HTTP 响应一点时间发出去，再替换进程。
	time.Sleep(150 * time.Millisecond)
	return syscall.Exec(exePath, os.Args, os.Environ())
}

// stopSelf 退出当前进程。
func stopSelf() {
	log.Printf("正在停止服务")
	time.Sleep(150 * time.Millisecond)
	os.Exit(0)
}

// serviceInfo 返回当前进程的运行信息。
func serviceInfo() map[string]any {
	exePath, _ := os.Executable()
	return map[string]any{
		"pid":       os.Getpid(),
		"exe":       exePath,
		"args":      os.Args,
		"startedAt": processStartTime(),
	}
}

// processStartTime 粗略估计进程启动时间（用当前时间减运行时长）。
// 由 adminState 记录的 start 提供，这里只作占位，实际由调用方填充。
func processStartTime() string {
	return ""
}

// handleService 挂载 /api/service/* 接口。
func handleService(mux *http.ServeMux, guard func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/service/info", guard(func(w http.ResponseWriter, r *http.Request) {
		writeAdminJSON(w, 200, map[string]any{"ok": true, "service": serviceInfo()})
	}))

	mux.HandleFunc("/api/service/restart", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeAdminJSON(w, 200, map[string]any{
			"ok":      true,
			"message": "正在重启，页面将在数秒后自动恢复",
		})
		go func() {
			if err := restartSelf(); err != nil {
				log.Printf("重启失败: %v", err)
			}
		}()
	}))

	mux.HandleFunc("/api/service/stop", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeAdminJSON(w, 200, map[string]any{
			"ok":      true,
			"message": "服务已停止，请手动重新启动",
		})
		go stopSelf()
	}))
}
