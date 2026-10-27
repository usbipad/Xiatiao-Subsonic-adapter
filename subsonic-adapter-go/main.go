package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// 入口：Subsonic 兼容网关。把 /rest/<端点> 转发为上游调用。
// 与 Node 版 index.ts 对齐，但音频流走代理并带超时保护。

var endpoints = map[string]func(routeCtx) routeResult{
	"ping":                 handlePing,
	"search3":              handleSearch3,
	"getSong":              handleGetSong,
	"getAlbum":             handleGetAlbum,
	"getArtist":            handleGetArtist,
	"getArtists":           handleGetArtists,
	"getAlbumList2":        handleGetAlbumList2,
	"getCoverArt":          handleGetCoverArt,
	"getPlaylists":         handleGetPlaylists,
	"getPlaylist":          handleGetPlaylist,
	"getRecommendations":   handleGetRecommendations,
	"stream":               handleStream,
	"getLyrics":            handleGetLyrics,
	"getLyricsBySongId":    handleGetLyricsBySongID,
	"getStarred2":          handleGetStarred2,
	"getRandomSongs":       handleGetRandomSongs,
	"star":                 handleStar,
	"unstar":               handleUnstar,
	"getUser":              handleGetUser,
	"getUserProfile":       handleGetUserProfile,
	"getAvatar":            handleGetAvatar,
	"scrobble":             handleScrobble,
}

func normalizeEndpoint(path string) string {
	if !strings.HasPrefix(path, "/rest/") {
		return ""
	}
	name := strings.TrimPrefix(path, "/rest/")
	name = strings.Trim(name, "/")
	if strings.Contains(name, "/") {
		return ""
	}
	return strings.TrimSuffix(name, ".view")
}

func writeJSON(w http.ResponseWriter, body map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_ = json.NewEncoder(w).Encode(body)
}

// proxyAudio 代理音频流：透传 Range，保证 206，带首字节/空闲超时。
// proxyAudio 代理音频流：透传 Range，保证 206，带首字节/空闲超时。
//
// refresh 用于在上游链接失效（vkey 过期、CDN 拒绝等）时重新换链，
// 返回新的直链 URL；为空表示无法换链。
func proxyAudio(w http.ResponseWriter, r *http.Request, rawURL string, refresh func() string) {
	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		http.Error(w, "bad upstream url", http.StatusBadGateway)
		return
	}
	if ua := r.Header.Get("User-Agent"); ua != "" {
		req.Header.Set("User-Agent", ua)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (subsonic-adapter-go)")
	}
	req.Header.Set("Referer", "https://y.qq.com/")
	clientRange := r.Header.Get("Range")
	if clientRange != "" {
		req.Header.Set("Range", clientRange)
	}

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 10 * time.Second,
			Proxy:                 http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("stream proxy 拉取失败: %v", err)
		http.Error(w, "upstream fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 只有「链接失效类」状态码才值得换链重试：
		//   401/403 —— 签名或 vkey 失效
		//   5xx     —— CDN 侧临时故障
		// 其余（如 416 Range Not Satisfiable、404）属于参数或资源问题，
		// 换链拿到的仍是同一个文件，重试只会造成无效请求风暴。
		retryable := resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode >= 500
		if retryable && refresh != nil {
			if fresh := refresh(); fresh != "" && fresh != rawURL {
				log.Printf("stream 直链失效（status=%d），已换链重试", resp.StatusCode)
				resp.Body.Close()
				proxyAudio(w, r, fresh, nil)
				return
			}
		}
		http.Error(w, fmt.Sprintf("upstream status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(resp.StatusCode)

	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			if rerr != io.EOF {
				log.Printf("stream proxy 传输中断: %v", rerr)
			}
			return
		}
	}
}

func main() {
	// 单实例保护：无论从桌面图标、systemd 还是命令行启动，全局只允许一个实例。
	// 否则会各自占一个端口（4535、4536…），既浪费也会让用户困惑。
	if _, ok := acquireInstanceLock(); !ok {
		// 已有实例在运行：对用户来说「再点一次图标」就是打开管理页，
		// 属于正常行为，不打印提示；只有打开浏览器失败时才在日志里留痕。
		openBrowserAndWait(fmt.Sprintf("http://127.0.0.1:%d", loadSavedPortOrDefault()))
		return
	}
	defer releaseInstanceLock()

	cfg := loadConfig()
	upstream := NewQQUpstream(cfg.UpstreamURL, cfg.UpstreamTimeout, cfg.UpstreamCookie)

	mux := http.NewServeMux()
	auth := newAuthStore()
	handleAuth(mux, auth)
	admin := newAdminState(cfg)
	admin.setAuth(auth)
	admin.handleAdmin(mux, upstream)
	handleService(mux, admin.requireAuth)
	handleAutostart(mux, admin.requireAuth)
	mux.HandleFunc("/rest/", func(w http.ResponseWriter, r *http.Request) {
		endpoint := normalizeEndpoint(r.URL.Path)
		if endpoint == "" {
			writeJSON(w, FailedResponse(EnvelopeOptions{Type: cfg.ServerType, ServerVersion: cfg.ServerVersion}, ErrMissingParameter, "endpoint required"))
			return
		}
		if !hasAuthParams(r.URL.Query()) {
			writeJSON(w, FailedResponse(EnvelopeOptions{Type: cfg.ServerType, ServerVersion: cfg.ServerVersion}, ErrAuthentication, "authentication required"))
			return
		}
		handler, ok := endpoints[endpoint]
		if !ok {
			writeJSON(w, FailedResponse(EnvelopeOptions{Type: cfg.ServerType, ServerVersion: cfg.ServerVersion}, ErrGeneric, "unsupported endpoint: "+endpoint))
			return
		}
		result := handler(routeCtx{query: r.URL.Query(), config: cfg, upstream: upstream})
		if result.proxyURL != "" {
			log.Printf("stream proxy id=%s range=%s", r.URL.Query().Get("id"), r.Header.Get("Range"))
			// 换链回调：清掉该曲目缓存后重新解析，返回新直链。
			refresh := func() string {
				q := r.URL.Query()
				res := handleStream(routeCtx{query: q, config: cfg, upstream: upstream, forceRefresh: true})
				return res.proxyURL
			}
			proxyAudio(w, r, result.proxyURL, refresh)
			return
		}
		if result.redirect != "" {
			w.Header().Set("Location", result.redirect)
			w.WriteHeader(result.status)
			return
		}
		// 仅当处理器显式指定了非 200 状态码时才用它。
		// 其余端点沿用 200 + 错误体的历史行为，避免影响既有客户端。
		if result.status > 0 && result.status != http.StatusOK && result.body != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(result.status)
			_ = json.NewEncoder(w).Encode(result.body)
			return
		}
		writeJSON(w, result.body)
	})

	// 显式通过 PORT 指定时，冲突就直接报错；否则允许自动顺延。
	portExplicit := strings.TrimSpace(os.Getenv("PORT")) != ""
	addr, ln, err := listenWithFallback(cfg.Host, cfg.Port, portExplicit)
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
	if ln != nil {
		cfg.Port = ln.Addr().(*net.TCPAddr).Port
	}
	// 记住本次实际使用的端口，下次启动优先复用，避免端口漂移。
	savePort(cfg.Port)

	log.Printf("subsonic-adapter-go listening on http://%s", addr)
	log.Printf("  upstream: %s", cfg.UpstreamURL)
	log.Printf("  管理页面: http://127.0.0.1:%d", cfg.Port)

	// 播放器连接地址：无图形环境下用户需要照抄这里填进 Subsonic 客户端。
	// 监听 0.0.0.0 时列出所有网卡地址；绑定具体地址时只列该地址。
	log.Printf("  播放器连接地址（Subsonic 服务器）:")
	if cfg.Host == "0.0.0.0" || cfg.Host == "::" {
		for _, ip := range localIPs() {
			log.Printf("    http://%s:%d", ip, cfg.Port)
		}
	} else {
		log.Printf("    http://%s:%d", cfg.Host, cfg.Port)
	}
	log.Printf("    用户名、密码任意")

	// 启动后自动打开管理页面（可用 NO_BROWSER=1 禁用）。
	go openBrowser(fmt.Sprintf("http://127.0.0.1:%d", cfg.Port))

	srv := &http.Server{Handler: mux}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
