package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// 管理页访问控制。
//
// 设计要点：
//   - 首次访问没有任何凭据时，引导用户设置用户名与密码（setup）。
//   - 之后所有管理请求都需要有效会话 Cookie。
//   - 密码只存 bcrypt 哈希，不保存明文。
//   - 会话 Cookie 用 HMAC 签名，含过期时间，避免伪造。
//   - /rest/* 不参与此认证：Subsonic 客户端用自己的 u/t/s/p 参数，
//     加了页面认证会直接破坏播放器连接。

const (
	authCookieName   = "subsonic_adapter_session"
	authSessionHours = 24 * 7
)

// authCredentials 是持久化到磁盘的凭据。
type authCredentials struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	Secret       string    `json:"secret"`
	CreatedAt    time.Time `json:"createdAt"`
}

// authStore 管理凭据文件的读写与会话签发。
type authStore struct {
	mu   sync.RWMutex
	path string
	cred *authCredentials
}

// authFilePath 返回凭据文件路径，放在配置目录下。
func authFilePath() string {
	dir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "subsonic-adapter", "auth.json")
}

// newAuthStore 加载已有凭据；没有则返回未配置状态。
func newAuthStore() *authStore {
	s := &authStore{path: authFilePath()}
	if s.path == "" {
		return s
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return s
	}
	var c authCredentials
	if json.Unmarshal(data, &c) != nil {
		return s
	}
	if c.Username == "" || c.PasswordHash == "" || c.Secret == "" {
		return s
	}
	s.cred = &c
	return s
}

// configured 判断是否已完成首次设置。
func (s *authStore) configured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cred != nil
}

// setup 完成首次设置；已配置时返回错误，避免被覆盖。
func (s *authStore) setup(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cred != nil {
		return fmt.Errorf("已完成初始化，不能重复设置")
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(password) < 4 {
		return fmt.Errorf("密码至少 4 位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码处理失败: %w", err)
	}
	secret, err := randomHex(32)
	if err != nil {
		return fmt.Errorf("生成密钥失败: %w", err)
	}
	c := &authCredentials{
		Username:     strings.TrimSpace(username),
		PasswordHash: string(hash),
		Secret:       secret,
		CreatedAt:    time.Now(),
	}
	if err := s.persistLocked(c); err != nil {
		return err
	}
	s.cred = c
	return nil
}

// currentUsername 返回当前用户名；未初始化返回空串。
func (s *authStore) currentUsername() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cred == nil {
		return ""
	}
	return s.cred.Username
}

// verify 校验用户名密码。
func (s *authStore) verify(username, password string) bool {
	s.mu.RLock()
	cred := s.cred
	s.mu.RUnlock()
	if cred == nil {
		return false
	}
	if cred.Username != strings.TrimSpace(username) {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)) == nil
}

// updateCredentials 修改用户名与密码（需要已登录）。
func (s *authStore) updateCredentials(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cred == nil {
		return fmt.Errorf("尚未初始化")
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(password) < 4 {
		return fmt.Errorf("密码至少 4 位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码处理失败: %w", err)
	}
	c := &authCredentials{
		Username:     strings.TrimSpace(username),
		PasswordHash: string(hash),
		Secret:       s.cred.Secret,
		CreatedAt:    s.cred.CreatedAt,
	}
	if err := s.persistLocked(c); err != nil {
		return err
	}
	s.cred = c
	return nil
}

// persistLocked 写入凭据文件（调用方需持有写锁）。
func (s *authStore) persistLocked(c *authCredentials) error {
	if s.path == "" {
		return fmt.Errorf("无法确定配置目录")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// 0600：只有本人可读写，避免凭据泄露。
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("写入凭据失败: %w", err)
	}
	return nil
}

// issueToken 生成签名会话令牌：payload.signature。
func (s *authStore) issueToken() (string, error) {
	s.mu.RLock()
	cred := s.cred
	s.mu.RUnlock()
	if cred == nil {
		return "", fmt.Errorf("尚未初始化")
	}
	exp := time.Now().Add(authSessionHours * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%d", cred.Username, exp)
	sig := signPayload(cred.Secret, payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig, nil
}

// validateToken 校验令牌签名与有效期。
func (s *authStore) validateToken(token string) bool {
	s.mu.RLock()
	cred := s.cred
	s.mu.RUnlock()
	if cred == nil || token == "" {
		return false
	}
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	payload := string(raw)
	if !hmac.Equal([]byte(signPayload(cred.Secret, payload)), []byte(parts[1])) {
		return false
	}
	seg := strings.SplitN(payload, "|", 2)
	if len(seg) != 2 || seg[0] != cred.Username {
		return false
	}
	var exp int64
	if _, err := fmt.Sscanf(seg[1], "%d", &exp); err != nil {
		return false
	}
	return time.Now().Unix() < exp
}

// signPayload 用 HMAC-SHA256 对负载签名。
func signPayload(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// randomHex 生成 n 字节随机数的十六进制串。
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sessionFromRequest 从 Cookie 读取会话令牌。
func sessionFromRequest(r *http.Request) string {
	c, err := r.Cookie(authCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// setSessionCookie 写入会话 Cookie。
func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   authSessionHours * 3600,
	})
}

// clearSessionCookie 清除会话 Cookie。
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
