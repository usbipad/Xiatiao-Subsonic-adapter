package main

import (
	"encoding/json"
	"net/http"
)

// 认证接口与中间件。

// handleAuth 挂载 /api/auth/* 接口。
func handleAuth(mux *http.ServeMux, store *authStore) {
	// 查询当前认证状态：是否已初始化、本次请求是否已登录。
	mux.HandleFunc("/api/auth/state", func(w http.ResponseWriter, r *http.Request) {
		loggedIn := store.validateToken(sessionFromRequest(r))
		writeAdminJSON(w, 200, map[string]any{
			"ok":         true,
			"configured": store.configured(),
			"loggedIn":   loggedIn,
			"username":   store.currentUsername(),
		})
	})

	// 首次设置用户名与密码。
	mux.HandleFunc("/api/auth/setup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": "参数错误"})
			return
		}
		if err := store.setup(body.Username, body.Password); err != nil {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		// 设置完成即自动登录，省去用户再输一次。
		token, err := store.issueToken()
		if err == nil {
			setSessionCookie(w, token)
		}
		writeAdminJSON(w, 200, map[string]any{"ok": true})
	})

	// 登录。
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": "参数错误"})
			return
		}
		if !store.verify(body.Username, body.Password) {
			writeAdminJSON(w, 401, map[string]any{"ok": false, "error": "用户名或密码错误"})
			return
		}
		token, err := store.issueToken()
		if err != nil {
			writeAdminJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		setSessionCookie(w, token)
		writeAdminJSON(w, 200, map[string]any{"ok": true})
	})

	// 退出登录。
	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		clearSessionCookie(w)
		writeAdminJSON(w, 200, map[string]any{"ok": true})
	})

	// 修改用户名密码（需已登录）。
	mux.HandleFunc("/api/auth/change", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !store.validateToken(sessionFromRequest(r)) {
			writeAdminJSON(w, 401, map[string]any{"ok": false, "error": "未登录"})
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": "参数错误"})
			return
		}
		if err := store.updateCredentials(body.Username, body.Password); err != nil {
			writeAdminJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		// 旧会话仍有效（密钥未变），但用户名变了会让旧令牌失效，重新签发。
		token, err := store.issueToken()
		if err == nil {
			setSessionCookie(w, token)
		}
		writeAdminJSON(w, 200, map[string]any{"ok": true})
	})
}

// requireAuth 包装受保护的管理接口：未登录返回 401。
// 只用于 /api/* 管理接口，不用于 /rest/*。
func requireAuth(store *authStore, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.validateToken(sessionFromRequest(r)) {
			writeAdminJSON(w, 401, map[string]any{"ok": false, "error": "未登录"})
			return
		}
		next(w, r)
	}
}
