package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 管理接口：与老的 Node 版管理页等价（上游地址、连通性、扫码登录、状态）。
// 不做用户登录校验。

type adminState struct {
	mu     sync.RWMutex
	config AdapterConfig
	start  time.Time
	auth   *authStore
}

// setAuth 注入认证存储，供受保护接口使用。
func (a *adminState) setAuth(s *authStore) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.auth = s
}

// requireAuth 对管理接口做登录校验。
// 未配置任何凭据时放行——否则首次访问将无法进入设置页，形成死锁。
func (a *adminState) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		store := a.auth
		a.mu.RUnlock()
		if store == nil || !store.configured() {
			// 尚未初始化：允许访问，页面会引导用户设置用户名密码。
			next(w, r)
			return
		}
		if !store.validateToken(sessionFromRequest(r)) {
			writeAdminJSON(w, 401, map[string]any{"ok": false, "error": "未登录"})
			return
		}
		next(w, r)
	}
}

func newAdminState(cfg AdapterConfig) *adminState {
	return &adminState{config: cfg, start: time.Now()}
}

func (a *adminState) snapshot() AdapterConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

func (a *adminState) setUpstream(url string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config.UpstreamURL = trimURL(url)
}

func writeAdminJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// handleAdmin 挂载所有 /api/* 管理接口与首页。
func (a *adminState) handleAdmin(mux *http.ServeMux, upstream *QQUpstream) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(adminPageHTML))
	})

	mux.HandleFunc("/api/status", a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		cfg := a.snapshot()
		ips := localIPs()
		addresses := make([]map[string]string, 0, len(ips))
		for _, ip := range ips {
			addresses = append(addresses, map[string]string{
				"ip":      ip,
				"url":     "http://" + ip + ":" + itoa(cfg.Port),
				"display": ip + ":" + itoa(cfg.Port),
			})
		}
		nickname, avatar, loggedIn := accountInfo(upstream)
		writeAdminJSON(w, 200, map[string]any{
			"ok":            true,
			"port":          cfg.Port,
			"host":          cfg.Host,
			"upstreamUrl":   cfg.UpstreamURL,
			"serverType":    cfg.ServerType,
			"serverVersion": cfg.ServerVersion,
			"uptimeSec":     int(time.Since(a.start).Seconds()),
			"loggedIn":      loggedIn,
			"nickname":      nickname,
			"avatar":        avatar,
			"localUrl":      "http://127.0.0.1:" + itoa(cfg.Port),
			"addresses":     addresses,
		})
	}))

	mux.HandleFunc("/api/upstream", a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			UpstreamURL string `json:"upstreamUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.UpstreamURL) == "" {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": "upstreamUrl required"})
			return
		}
		a.setUpstream(body.UpstreamURL)
		upstream.baseURL = trimURL(body.UpstreamURL)
		writeAdminJSON(w, 200, map[string]any{"ok": true, "upstreamUrl": upstream.baseURL})
	}))

	mux.HandleFunc("/api/upstream/check", a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		reachable := checkUpstreamReachable(upstream)
		writeAdminJSON(w, 200, map[string]any{"ok": true, "reachable": reachable})
	}))

	// 上游登录通道；顺序即页面展示顺序。
	mux.HandleFunc("/api/login/channels", a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		writeAdminJSON(w, 200, map[string]any{
			"ok":       true,
			"channels": []map[string]string{
				{"id": "qq", "name": "QQ"},
				{"id": "wechat", "name": "微信"},
				{"id": "qqmusic", "name": "QQ 音乐"},
			},
		})
	}))

	mux.HandleFunc("/api/qrcode", a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		channel := strings.TrimSpace(r.URL.Query().Get("channel"))
		key, ok := upstream.GetQrKey(channel)
		if !ok {
			writeAdminJSON(w, 200, map[string]any{"ok": false, "error": "获取二维码失败"})
			return
		}
		img, ok := upstream.GetQrImage(key)
		if !ok {
			writeAdminJSON(w, 200, map[string]any{"ok": false, "error": "生成二维码失败"})
			return
		}
		writeAdminJSON(w, 200, map[string]any{"ok": true, "key": key, "qrimg": img, "channel": channel})
	}))

	mux.HandleFunc("/api/login/check", a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": "key required"})
			return
		}
		code, token := upstream.CheckQr(key)
		if token != "" {
			upstream.SetSession(token)
		}
		writeAdminJSON(w, 200, map[string]any{"ok": true, "code": code, "loggedIn": token != ""})
	}))

	mux.HandleFunc("/api/logout", a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		upstream.SetSession("")
		writeAdminJSON(w, 200, map[string]any{"ok": true})
	}))
}

// checkLoggedIn 用 user/detail 判断上游登录态。
func checkLoggedIn(u *QQUpstream) bool {
	data := u.Get("/user/detail", nil)
	if data == nil {
		return false
	}
	return idOf(firstNonNil(dict(data)["profile"], nil)) != "" || dict(data)["code"] != nil
}

// accountInfo 读取当前登录账号信息。未登录或读取失败时返回零值。
// 上游返回结构：{code:200, profile:{musicid, info:{nick, logo, ...}}}
// 昵称与头像都在 profile.info 下，外层没有 nickname 字段。
func accountInfo(u *QQUpstream) (nickname, avatar string, loggedIn bool) {
	data := u.Get("/user/detail", nil)
	if data == nil {
		return "", "", false
	}
	profile := dict(dict(data)["profile"])
	if len(profile) == 0 {
		return "", "", false
	}
	info := dict(profile["info"])
	nickname = firstText(info["nick"], info["nickname"], profile["nickname"], profile["nick"])
	avatar = firstText(info["logo"], profile["logo"])
	return nickname, avatar, true
}

// checkUpstreamReachable 只判断上游服务是否可达，与登录态无关。
// 必须使用无需鉴权的公开端点：/user/detail 在未登录时返回 401，
// 用它探活会把「未登录」误报成「上游不可达」。
func checkUpstreamReachable(u *QQUpstream) bool {
	resp, err := u.GetRaw("/login/channels", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
