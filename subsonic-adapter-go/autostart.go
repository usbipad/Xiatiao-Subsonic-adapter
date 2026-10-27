package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 登录自启管理。
//
// 策略：优先使用 systemd 用户服务（可靠、崩溃自动重启、日志可查），
// systemd 不可用或未提供用户服务单元时，降级为 XDG autostart。
// 两种机制互斥，同一时刻只用一种，避免登录后重复启动。

const autostartFileName = "subsonic-adapter.desktop"
const systemdUnitName = "subsonic-adapter.service"

// abbrevHome 把家目录前缀替换成 ~，便于展示。
func abbrevHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return "~" + path[len(home):]
	}
	return path
}

// autostartPath 返回 XDG autostart 条目的完整路径。
func autostartPath() string {
	dir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", autostartFileName)
}

// autostartFileExists 判断 XDG autostart 条目是否存在。
func autostartFileExists() bool {
	path := autostartPath()
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// autostartDesktopContent 生成 autostart 条目内容（使用 --no-browser）。
func autostartDesktopContent(launchScript string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Subsonic Adapter (Autostart)
Comment=Start Subsonic Adapter in background on login
Exec=%s --no-browser
X-GNOME-Autostart-enabled=true
NoDisplay=true
Terminal=false
`, launchScript)
}

// enableAutostart 写入 XDG autostart 条目。
func enableAutostart(launchScript string) error {
	path := autostartPath()
	if path == "" {
		return fmt.Errorf("无法确定配置目录")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(autostartDesktopContent(launchScript)), 0o644); err != nil {
		return fmt.Errorf("写入自启文件失败: %w", err)
	}
	return nil
}

// disableAutostart 删除 XDG autostart 条目。
func disableAutostart() error {
	path := autostartPath()
	if path == "" {
		return fmt.Errorf("无法确定配置目录")
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除自启文件失败: %w", err)
	}
	return nil
}

// systemctlTimeout 限制单次 systemctl 调用时长。
// 没有超时的话，D-Bus 卡死会让 HTTP 处理函数永久阻塞，页面转圈不止。
const systemctlTimeout = 5 * time.Second

// runSystemctl 执行 systemctl --user，带超时保护。
func runSystemctl(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlTimeout)
	defer cancel()
	all := append([]string{"--user"}, args...)
	return exec.CommandContext(ctx, "systemctl", all...).Output()
}

// systemdUserAvailable 判断当前环境是否有可用的 systemd 用户会话，
// 且系统里确实存在我们的用户服务单元。
func systemdUserAvailable() bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	// 需要有用户级 systemd 会话（XDG_RUNTIME_DIR 下的 bus 或 user@.service）。
	if strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")) == "" {
		return false
	}
	// 必须真的能查询到该单元，否则 enable 会失败。
	out, err := runSystemctl("list-unit-files", systemdUnitName, "--no-legend", "--no-pager")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), systemdUnitName)
}

// systemdUnitEnabled 判断 systemd 用户服务是否已启用（开机/登录自启）。
func systemdUnitEnabled() bool {
	if !systemdUserAvailable() {
		return false
	}
	out, err := runSystemctl("is-enabled", systemdUnitName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "enabled"
}

// systemdUnitActive 判断 systemd 用户服务当前是否正在运行。
func systemdUnitActive() bool {
	if !systemdUserAvailable() {
		return false
	}
	out, err := runSystemctl("is-active", systemdUnitName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// systemdEnable 启用并启动用户服务。
func systemdEnable() error {
	if _, err := runSystemctl("enable", systemdUnitName); err != nil {
		return fmt.Errorf("systemctl enable 失败: %w", err)
	}
	// 启动失败不致命（可能已有实例在跑），仅报告 enable 结果。
	_, _ = runSystemctl("start", systemdUnitName)
	return nil
}

// systemdDisable 停用用户服务。
func systemdDisable() error {
	_, _ = runSystemctl("stop", systemdUnitName)
	if _, err := runSystemctl("disable", systemdUnitName); err != nil {
		return fmt.Errorf("systemctl disable 失败: %w", err)
	}
	return nil
}

// launchScriptNear 返回可用的 launch.sh 路径；找不到则返回空。
//
// 查找顺序：
//  1. 与可执行文件同目录（源码目录直接运行时的情形）；
//  2. /usr/share/subsonic-adapter/launch.sh（发行包安装位置）。
//
// 第二个位置是必需的：安装到 /usr/bin 后，同目录下不会有 launch.sh。
func launchScriptNear() string {
	candidates := []string{}

	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "launch.sh"))
	}
	candidates = append(candidates, "/usr/share/subsonic-adapter/launch.sh")

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// autostartMethod 返回当前生效的自启方式：systemd / xdg / none。
func autostartMethod() string {
	if systemdUnitEnabled() {
		return "systemd"
	}
	if autostartFileExists() {
		return "xdg"
	}
	return "none"
}

// autostartLocation 返回当前（或默认）自启配置的可读位置。
func autostartLocation() string {
	if systemdUnitEnabled() {
		return "systemd 用户服务（systemctl --user）"
	}
	if p := autostartPath(); p != "" {
		return abbrevHome(p)
	}
	return ""
}

// handleAutostart 挂载 /api/autostart 接口。
func handleAutostart(mux *http.ServeMux, guard func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/autostart", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeAdminJSON(w, 200, map[string]any{
				"ok":       true,
				"enabled":  autostartMethod() != "none",
				"method":   autostartMethod(),
				"location": autostartLocation(),
				"systemdSupported": systemdUserAvailable(),
			})
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": "参数错误"})
			return
		}

		if body.Enabled {
			// 优先 systemd：可用则用它，并清除可能遗留的 XDG 条目，避免双启。
			if systemdUserAvailable() {
				if err := systemdEnable(); err != nil {
					writeAdminJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
					return
				}
				_ = disableAutostart()
			} else {
				launch := launchScriptNear()
				if launch == "" {
					writeAdminJSON(w, 400, map[string]any{
						"ok":    false,
						"error": "当前环境无 systemd 用户会话，且未找到 launch.sh，无法设置自启",
					})
					return
				}
				if err := enableAutostart(launch); err != nil {
					writeAdminJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
					return
				}
			}
		} else {
			// 关闭：两种机制都清掉，确保干净。
			if systemdUserAvailable() {
				_ = systemdDisable()
			}
			if err := disableAutostart(); err != nil {
				writeAdminJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
		}

		writeAdminJSON(w, 200, map[string]any{
			"ok":       true,
			"enabled":  autostartMethod() != "none",
			"method":   autostartMethod(),
			"location": autostartLocation(),
		})
	}))
}
