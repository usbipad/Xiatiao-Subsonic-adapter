package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"
)

// 播放、歌词、收藏、用户端点。对应 Node 版 playback.ts。

var numericIDPattern = regexp.MustCompile(`^\d+$`)

// songHints 榜单等来源的数字 id → {title, artist} 内存映射。
type songHint struct {
	Title  string
	Artist string
}

// songHintsMu 保护 songHints：HTTP handler 并发执行，榜单加载（写）
// 与 stream 解析（读）可能同时发生。无锁读写普通 map 会触发 Go 运行时
// "concurrent map read and map write" 直接 panic，故必须加锁。
var (
	songHintsMu sync.RWMutex
	songHints   = map[string]songHint{}
)

func rememberSong(numericID string, hint songHint) {
	songHintsMu.Lock()
	defer songHintsMu.Unlock()
	if len(songHints) >= 5000 {
		// 淘汰最早写入的一批：Go map 无序，这里按「全量清空重建」
		// 的简化策略避免随机删掉热点。上限 5000 条约 500KB，可控。
		if len(songHints) >= 6000 {
			songHints = make(map[string]songHint, 5000)
		}
	}
	songHints[numericID] = hint
}

func recallSong(numericID string) (songHint, bool) {
	songHintsMu.RLock()
	defer songHintsMu.RUnlock()
	h, ok := songHints[numericID]
	return h, ok
}

// resolvePlayableID 纯数字 id 用缓存的歌名精确搜索换出 mid。
func resolvePlayableID(c routeCtx, id string) string {
	if !numericIDPattern.MatchString(id) {
		return id
	}
	hint, ok := recallSong(id)
	if !ok {
		return id
	}
	keyword := hint.Artist + " " + hint.Title
	data := c.upstream.Get("/search", map[string]string{
		"keyword": keyword, "type": "song", "page": "1", "limit": "10",
	})
	items, _ := dict(data)["items"].([]any)
	for _, entry := range items {
		item := dict(entry)
		if idOf(item["id"]) == id {
			if mid := idOf(firstNonNil(item["mid"], item["songmid"])); mid != "" {
				return mid
			}
		}
	}
	return id
}

func mapQuality(quality string, maxBitRate int) string {
	switch quality {
	case "standard":
		return "128"
	case "high":
		return "320"
	case "lossless", "hires", "master":
		return "flac"
	}
	if maxBitRate <= 0 {
		return "flac"
	}
	if maxBitRate <= 128 {
		return "128"
	}
	if maxBitRate <= 320 {
		return "320"
	}
	return "flac"
}

func handleStream(c routeCtx) routeResult {
	rawID := strOf(c.query, "id")
	if rawID == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	id := resolvePlayableID(c, rawID)
	quality := strOf(c.query, "quality")
	maxBitRate := intOf(c.query, "maxBitRate", 0)
	upstreamQuality := mapQuality(quality, maxBitRate)

	// 命中直链缓存：避免每次 seek 都重新换链（上游换链有频率限制）。
	cacheKey := id + ":" + upstreamQuality
	if !c.forceRefresh {
		if cached, ok := globalStreamCache.get(cacheKey); ok {
			return routeResult{status: 200, proxyURL: cached}
		}
	} else {
		// 强制换链：先清掉可能已过期的旧条目。
		globalStreamCache.invalidate(cacheKey)
	}

	data := c.upstream.Get("/getMusicPlay", map[string]string{"songmid": id, "quality": upstreamQuality})
	playURL := dict(dict(dict(data)["data"])["playUrl"])
	entry := dict(playURL[id])
	url, _ := entry["url"].(string)
	if url == "" {
		// stream 是二进制端点。上游明确表示拿不到播放地址时（通常是版权/
		// 会员限制），若仍返回 200 + JSON 错误体，客户端会把这段 JSON 当作
		// 音频数据解码：symphonia 要求 206，直接判失败；ffmpeg 探测不到音频流
		// 也失败；最终只能靠客户端的「卡住检测」超时兜底跳过（本项目为 25 秒）。
		//
		// 实测：status=200 + 177 字节 JSON 时，播放器确实要等 25 秒才切歌。
		// 返回 404 后，客户端在 HTTP 层即可立即判定失败。
		// 其他 JSON 端点不受影响，仍保持 200 + 错误体。
		return routeResult{status: http.StatusNotFound, body: FailedResponse(c.env(), ErrAuthorization, "stream unavailable")}
	}
	globalStreamCache.set(cacheKey, url)
	return routeResult{status: 200, proxyURL: url}
}

func handleGetLyricsBySongID(c routeCtx) routeResult {
	id := strOf(c.query, "id")
	if id == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	data := c.upstream.Get("/getLyric", map[string]string{"songmid": id})
	lyric, _ := dict(dict(data)["response"])["lyric"].(string)
	lyrics := structuredLyricsOf(lyric, "")
	return c.ok(map[string]any{"lyricsList": map[string]any{"structuredLyrics": lyrics}})
}

func handleGetLyrics(c routeCtx) routeResult {
	artist := strOf(c.query, "artist")
	title := strOf(c.query, "title")
	lyrics := map[string]any{"value": ""}
	if artist != "" {
		lyrics["artist"] = artist
	}
	if title != "" {
		lyrics["title"] = title
	}
	return c.ok(map[string]any{"lyrics": lyrics})
}

func handleGetStarred2(c routeCtx) routeResult {
	data := c.upstream.Get("/user/liked-songs", map[string]string{"offset": "0", "limit": "100"})
	songs := songsOf(dict(data)["songs"])
	return c.ok(map[string]any{"starred2": map[string]any{"song": songsJSON(songs)}})
}

func handleGetRandomSongs(c routeCtx) routeResult {
	size := intOf(c.query, "size", 10)
	if size > 20 {
		size = 20
	}
	if size < 1 {
		size = 1
	}
	data := c.upstream.Get("/search", map[string]string{
		"keyword": "热门", "type": "song", "page": "1", "limit": itoa(size),
	})
	songs := songsOf(searchItemsOf(data))
	return c.ok(map[string]any{"randomSongs": map[string]any{"song": songsJSON(songs)}})
}

// numericIDOf 用 mid 查上游详情拿数字 id。
func numericIDOf(c routeCtx, mid string) (int, bool) {
	data := c.upstream.Get("/getSongInfo", map[string]string{"songmid": mid})
	track := dict(dict(dict(dict(data)["response"])["songinfo"])["data"])["track_info"]
	id, ok := num(dict(track)["id"])
	if !ok || id <= 0 {
		return 0, false
	}
	return id, true
}

func setLiked(c routeCtx, liked bool) routeResult {
	mid := strOf(c.query, "id")
	if mid == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	// 上游写接口需要数字 id 与 mid 同时提供。
	numericID, ok := numericIDOf(c, mid)
	if !ok {
		return c.fail(ErrNotFound, "song not found")
	}

	// 上游 /user/song/like 只接受 POST + JSON body：
	//   { "song": { "id": <number>, "mid": "<mid>", "type": 0 }, "liked": <bool> }
	// type 固定为 0（单曲）。
	payload := map[string]any{
		"song": map[string]any{
			"id":   numericID,
			"mid":  mid,
			"type": 0,
		},
		"liked": liked,
	}
	resp := c.upstream.PostJSON("/user/song/like", payload)
	if resp == nil {
		// 失败时明确报错，避免前端乐观更新后与服务端长期不一致。
		return c.fail(ErrGeneric, "failed to update like status")
	}
	if code, ok := num(dict(resp)["code"]); ok && code >= 400 {
		return c.fail(ErrGeneric, firstText(dict(resp)["message"], "failed to update like status"))
	}
	return c.ok(map[string]any{})
}

func handleStar(c routeCtx) routeResult   { return setLiked(c, true) }
func handleUnstar(c routeCtx) routeResult { return setLiked(c, false) }

func handleGetUser(c routeCtx) routeResult {
	username := strOf(c.query, "username")
	if username == "" {
		username = strOf(c.query, "u")
	}
	return c.ok(map[string]any{"user": map[string]any{
		"username": username, "scrobblingEnabled": true, "adminRole": true,
		"downloadRole": true, "uploadRole": false, "playlistRole": true,
		"coverArtRole": true, "streamRole": true, "shareRole": true, "jukeboxRole": false,
	}})
}

func handleScrobble(c routeCtx) routeResult { return c.ok(map[string]any{}) }

func handleGetUserProfile(c routeCtx) routeResult {
	data := c.upstream.Get("/user/detail", nil)
	profile := dict(dict(data)["profile"])
	info := dict(profile["info"])
	nickname, _ := info["nick"].(string)
	avatar, _ := info["logo"].(string)
	if avatar == "" {
		avatar, _ = profile["avatarUrl"].(string)
	}
	username := strOf(c.query, "username")
	if username == "" {
		username = strOf(c.query, "u")
	}
	return c.ok(map[string]any{"userProfile": map[string]any{
		"username": username, "nickname": nickname, "avatarUrl": avatar,
		"isVip": false, "source": "upstream",
	}})
}

func handleGetAvatar(c routeCtx) routeResult {
	data := c.upstream.Get("/user/detail", nil)
	profile := dict(dict(data)["profile"])
	info := dict(profile["info"])
	avatar, _ := info["logo"].(string)
	if avatar == "" {
		avatar, _ = profile["avatarUrl"].(string)
	}
	if avatar == "" {
		return c.fail(ErrNotFound, "no avatar")
	}
	return routeResult{status: 302, redirect: avatar}
}

func itoa(n int) string { return strconv.Itoa(n) }

var _ = fmt.Sprintf
