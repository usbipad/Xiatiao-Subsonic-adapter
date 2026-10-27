package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// QQUpstream 上游 HTTP 客户端：只通过 HTTP 调上游公开端点，与上游实现解耦。
// 对应 Node 版 upstream.ts。
type QQUpstream struct {
	baseURL      string
	timeout      time.Duration
	sessionToken string
	cookie       string
	httpClient   *http.Client
}

// sessionFilePath 返回会话文件路径。
//
// 必须使用绝对路径：程序可能由桌面图标、systemd、命令行等不同方式启动，
// 工作目录各不相同；用相对路径会导致「明明登录过，重启后却读不到会话」。
func sessionFilePath() string {
	dir := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			// 无法确定家目录时退回当前目录，保证功能不因此完全失效。
			return ".adapter-session"
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "subsonic-adapter", "session")
}

// legacySessionCandidates 返回旧版会话文件可能的位置。
//
// 早期版本把会话存成相对路径 `.adapter-session`，实际落点取决于启动时的工作目录：
// 从项目目录运行就在项目目录，从桌面图标运行就在 $HOME。
// 迁移时必须把这些位置都试一遍，否则会「升级后依然显示未登录」。
func legacySessionCandidates() []string {
	paths := []string{".adapter-session"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths, filepath.Join(home, ".adapter-session"))
	}
	if exePath, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exePath), ".adapter-session"))
	}
	return paths
}

func loadPersistedToken() string {
	path := sessionFilePath()
	data, err := os.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(data))
	}

	// 兼容旧位置：逐个尝试，找到即迁移到新位置。
	for _, legacy := range legacySessionCandidates() {
		raw, lerr := os.ReadFile(legacy)
		if lerr != nil {
			continue
		}
		token := strings.TrimSpace(string(raw))
		if token == "" {
			continue
		}
		persistToken(token)
		return token
	}
	return ""
}

func persistToken(token string) {
	path := sessionFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("无法创建会话目录: %v", err)
		return
	}
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		log.Printf("无法写入会话文件: %v", err)
	}
}

// NewQQUpstream 创建上游客户端。
func NewQQUpstream(baseURL string, timeout time.Duration, cookie string) *QQUpstream {
	return &QQUpstream{
		baseURL:      trimURL(baseURL),
		timeout:      timeout,
		sessionToken: loadPersistedToken(),
		cookie:       cookie,
		httpClient:   &http.Client{Timeout: timeout},
	}
}

func (u *QQUpstream) buildURL(path string, params map[string]string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	s := u.baseURL + path
	if encoded := q.Encode(); encoded != "" {
		s += "?" + encoded
	}
	return s
}

func (u *QQUpstream) headers() map[string]string {
	h := map[string]string{"Accept": "application/json"}
	cookies := []string{}
	if u.sessionToken != "" {
		cookies = append(cookies, "qqmusic_session="+u.sessionToken)
	}
	if u.cookie != "" {
		cookies = append(cookies, u.cookie)
	}
	if len(cookies) > 0 {
		h["Cookie"] = strings.Join(cookies, "; ")
	}
	return h
}

// Get 请求上游并解析 JSON；失败返回 nil（与 Node 版语义一致）。
func (u *QQUpstream) Get(path string, params map[string]string) map[string]any {
	req, err := http.NewRequest(http.MethodGet, u.buildURL(path, params), nil)
	if err != nil {
		return nil
	}
	for k, v := range u.headers() {
		req.Header.Set(k, v)
	}
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	return out
}

// GetRaw 返回原始响应（供代理/重定向场景）。
func (u *QQUpstream) GetRaw(path string, params map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u.buildURL(path, params), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range u.headers() {
		req.Header.Set(k, v)
	}
	return u.httpClient.Do(req)
}

// PostJSON 以 JSON 请求体 POST 到上游，并解析 JSON 响应。
//
// 上游部分写操作（如「我喜欢」的 /user/song/like）只接受 POST +
// JSON body，GET 语义无法表达，因此单独提供此方法。
// 失败返回 nil，与 Get 的语义保持一致。
func (u *QQUpstream) PostJSON(path string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	req, err := http.NewRequest(http.MethodPost, u.buildURL(path, nil), bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range u.headers() {
		req.Header.Set(k, v)
	}
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	return out
}

// SetSession 更新会话 token 并持久化。
func (u *QQUpstream) SetSession(token string) {
	u.sessionToken = token
	persistToken(token)
}

var _ = fmt.Sprintf
var _ = filepath.Join
