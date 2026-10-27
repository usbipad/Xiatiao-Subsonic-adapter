package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// hasGraphicalSession 判断当前是否有可用的图形会话。
//
// 无图形环境（纯 CLI 服务器、SSH、systemd system 服务）下不应尝试打开浏览器：
// 命令会静默失败，日志还会误报「已打开」，对排查毫无帮助。
func hasGraphicalSession() bool {
	if strings.TrimSpace(os.Getenv("DISPLAY")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) != "" {
		return true
	}
	// 部分环境只给 XDG_RUNTIME_DIR，wayland-0 socket 存在即视为有会话。
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir != "" {
		if _, err := os.Stat(filepath.Join(runtimeDir, "wayland-0")); err == nil {
			return true
		}
	}
	return false
}

// ensureSessionEnv 补全从桌面环境启动时常常缺失的会话变量。
// 从文件管理器双击启动的进程，环境比终端里少很多，Wayland/X11 的打开命令会因此失败。
func ensureSessionEnv() {
	setIfEmpty := func(key, value string) {
		if value == "" {
			return
		}
		if strings.TrimSpace(os.Getenv(key)) == "" {
			_ = os.Setenv(key, value)
		}
	}

	uid := strconv.Itoa(os.Getuid())
	runtimeDir := "/run/user/" + uid

	// 会话总线：xdg-open / gio 在 Wayland 下需要它来找到桌面门户。
	setIfEmpty("XDG_RUNTIME_DIR", runtimeDir)
	setIfEmpty("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(runtimeDir, "bus"))

	// 显示服务：优先 Wayland，退回 X11。
	if strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) == "" {
		if _, err := os.Stat(filepath.Join(runtimeDir, "wayland-0")); err == nil {
			_ = os.Setenv("WAYLAND_DISPLAY", "wayland-0")
		}
	}
	setIfEmpty("DISPLAY", ":0")
}

// openBrowserAndWait 与 openBrowser 相同，但会等待命令返回后再继续。
// 用于「进程即将退出」的场景：必须确保浏览器真的被拉起，否则用户点图标将毫无反应。
// 成功时静默；失败才写日志。
func openBrowserAndWait(url string) {
	if strings.TrimSpace(os.Getenv("NO_BROWSER")) == "1" {
		// 明确要求不开浏览器时，不打扰用户，也不写日志。
		return
	}
	if !hasGraphicalSession() {
		// 无图形会话：静默跳过。用户本来就是从终端启动的，不需要额外提示。
		return
	}
	ensureSessionEnv()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		if p, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command(p, url)
		} else if p, err := exec.LookPath("gio"); err == nil {
			cmd = exec.Command(p, "open", url)
		} else {
			log.Printf("未找到 xdg-open / gio，无法打开管理页。请手动访问 %s", url)
			return
		}
	}

	cmd.Env = os.Environ()
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			log.Printf("打开管理页失败: %v: %s（请手动访问 %s）", err, msg, url)
		} else {
			log.Printf("打开管理页失败: %v（请手动访问 %s）", err, url)
		}
	}
}

// openBrowser 尽力在默认浏览器中打开 url。
// 失败只记日志，不影响服务本身。
// 设置 NO_BROWSER=1 可禁用（适合 systemd / 后台运行）。
func openBrowser(url string) {
	if strings.TrimSpace(os.Getenv("NO_BROWSER")) == "1" {
		log.Printf("NO_BROWSER=1，跳过自动打开浏览器。请手动访问 %s", url)
		return
	}
	if !hasGraphicalSession() {
		// 无图形会话（CLI/SSH/服务器）：不尝试打开浏览器，避免误报成功。
		log.Printf("未检测到图形会话，跳过自动打开浏览器。请手动访问 %s", url)
		return
	}

	// 等服务完成监听，避免浏览器抢先打开拿到连接失败页。
	time.Sleep(300 * time.Millisecond)
	ensureSessionEnv()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		// Linux / BSD：按优先级挑选可用的打开方式。
		if p, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command(p, url)
		} else if p, err := exec.LookPath("gio"); err == nil {
			cmd = exec.Command(p, "open", url)
		} else {
			log.Printf("未找到 xdg-open / gio，请手动访问 %s", url)
			return
		}
	}

	// 继承补全后的环境，而不是父进程那份不完整的。
	cmd.Env = os.Environ()

	// 捕获输出用于诊断：xdg-open 出错往往只写 stderr 就退出。
	var stderr strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = nil

	if err := cmd.Start(); err != nil {
		log.Printf("自动打开浏览器失败: %v（请手动访问 %s）", err, url)
		return
	}

	// 异步等待，若命令非零退出则记录原因，方便排查。
	go func() {
		if err := cmd.Wait(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
				log.Printf("打开浏览器命令失败: %v: %s", err, msg)
			} else {
				log.Printf("打开浏览器命令失败: %v", err)
			}
			log.Printf("请手动访问 %s", url)
			return
		}
		log.Printf("已在浏览器中打开管理页面: %s", url)
	}()
}
