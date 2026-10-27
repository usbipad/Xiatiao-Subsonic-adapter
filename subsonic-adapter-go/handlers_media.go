package main

import (
	"net/url"
	"strings"
)

// 媒体端点：连通、搜索、单曲、专辑、艺术家、专辑列表、封面。对应 Node 版 media.ts。

// routeCtx 处理器上下文。
type routeCtx struct {
	query    url.Values
	config   AdapterConfig
	upstream *QQUpstream
	// forceRefresh 为真时跳过直链缓存，强制向上游重新换链。
	// 用于「上游直链已过期」的重试路径。
	forceRefresh bool
}

// routeResult 处理器结果：JSON 响应 / 重定向 / 代理音频。
type routeResult struct {
	status   int
	headers  map[string]string
	body     map[string]any
	redirect string
	proxyURL string
}

func intOf(q url.Values, key string, fallback int) int {
	v := strings.TrimSpace(q.Get(key))
	if v == "" {
		return fallback
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func strOf(q url.Values, key string) string {
	return strings.TrimSpace(q.Get(key))
}

func (c routeCtx) env() EnvelopeOptions {
	return EnvelopeOptions{Type: c.config.ServerType, ServerVersion: c.config.ServerVersion}
}

func (c routeCtx) ok(payload map[string]any) routeResult {
	return routeResult{status: 200, body: OkResponse(c.env(), payload)}
}

func (c routeCtx) fail(code int, message string) routeResult {
	return routeResult{status: 200, body: FailedResponse(c.env(), code, message)}
}

// searchItemsOf 从上游搜索结果中提取条目数组。
func searchItemsOf(data map[string]any) []any {
	root := data
	for _, key := range []string{"items", "item_song", "songlist", "song"} {
		if v, ok := root[key].([]any); ok {
			return v
		}
		if v, ok := dict(root["body"])[key].([]any); ok {
			return v
		}
	}
	return nil
}

// trackInfoOf 上游单曲详情里的 track_info。
func trackInfoOf(data map[string]any) any {
	return dict(dict(dict(dict(data)["response"])["songinfo"])["data"])["track_info"]
}

func handlePing(c routeCtx) routeResult { return c.ok(map[string]any{}) }

func handleSearch3(c routeCtx) routeResult {
	query := strOf(c.query, "query")
	if query == "" {
		return c.ok(map[string]any{"searchResult3": map[string]any{}})
	}
	songCount := intOf(c.query, "songCount", 50)
	songOffset := intOf(c.query, "songOffset", 0)
	page := songOffset/maxInt(1, songCount) + 1
	limit := songCount
	if limit > 20 {
		limit = 20
	}
	if limit < 1 {
		limit = 1
	}
	data := c.upstream.Get("/search", map[string]string{
		"keyword": query, "type": "song", "page": itoa(page), "limit": itoa(limit),
	})
	songs := songsOf(searchItemsOf(data))
	payload := map[string]any{}
	if len(songs) > 0 {
		payload["song"] = songsJSON(songs)
	}
	return c.ok(map[string]any{"searchResult3": payload})
}

func handleGetSong(c routeCtx) routeResult {
	id := strOf(c.query, "id")
	if id == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	data := c.upstream.Get("/getSongInfo", map[string]string{"songmid": id})
	song := songOf(trackInfoOf(data))
	if song == nil {
		return c.fail(ErrNotFound, "not found")
	}
	return c.ok(map[string]any{"song": songJSON(*song)})
}

func handleGetAlbum(c routeCtx) routeResult {
	id := strOf(c.query, "id")
	if id == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	data := c.upstream.Get("/getAlbumInfo", map[string]string{"albummid": id})
	info := dict(dict(dict(data)["response"])["data"])
	var list []any
	if v, ok := info["list"].([]any); ok {
		list = v
	} else if v, ok := info["songlist"].([]any); ok {
		list = v
	}
	albumMid := id
	if s, ok := info["albumMid"].(string); ok && s != "" {
		albumMid = s
	}
	songs := songsOf(list)
	for i := range songs {
		if songs[i].CoverArt == "" {
			songs[i].CoverArt = albumMid
		}
	}
	name := albumMid
	if s, ok := info["albumName"].(string); ok && s != "" {
		name = s
	}
	album := map[string]any{"id": albumMid, "name": name, "coverArt": albumMid, "songCount": len(songs)}
	if s, ok := info["singerName"].(string); ok {
		album["artist"] = s
	}
	if len(songs) > 0 {
		album["song"] = songsJSON(songs)
	}
	return c.ok(map[string]any{"album": album})
}

func handleGetArtist(c routeCtx) routeResult {
	id := strOf(c.query, "id")
	if id == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	return c.ok(map[string]any{"artist": map[string]any{"id": id, "name": id, "albumCount": 0}})
}

func handleGetArtists(c routeCtx) routeResult {
	return c.ok(map[string]any{"artists": map[string]any{"index": []any{}}})
}

func handleGetAlbumList2(c routeCtx) routeResult {
	size := intOf(c.query, "size", 10)
	if size > 20 {
		size = 20
	}
	if size < 1 {
		size = 1
	}
	data := c.upstream.Get("/recommend/new-albums", map[string]string{"limit": itoa(size)})
	var list []any
	if v, ok := dict(data)["albums"].([]any); ok {
		list = v
	}
	albums := []map[string]any{}
	for _, entry := range list {
		album := dict(entry)
		mid := idOf(firstNonNil(album["mid"], album["albummid"]))
		if mid == "" {
			continue
		}
		item := map[string]any{"id": mid, "name": mid, "coverArt": mid}
		if s, ok := album["name"].(string); ok && s != "" {
			item["name"] = s
		}
		if singers, ok := album["singers"].([]any); ok {
			names := []string{}
			for _, s := range singers {
				if n, ok := dict(s)["name"].(string); ok && n != "" {
					names = append(names, n)
				}
			}
			if len(names) > 0 {
				item["artist"] = strings.Join(names, " / ")
			}
		}
		albums = append(albums, item)
	}
	return c.ok(map[string]any{"albumList2": map[string]any{"album": albums}})
}

func handleGetCoverArt(c routeCtx) routeResult {
	id := strOf(c.query, "id")
	if id == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	size := intOf(c.query, "size", 300)
	if size == 0 {
		size = 300
	}
	prefix := "T002"
	if strings.HasPrefix(id, "top_") {
		prefix = "T003"
		id = strings.TrimPrefix(id, "top_")
	} else if strings.HasPrefix(id, "singer_") {
		prefix = "T001"
		id = strings.TrimPrefix(id, "singer_")
	}
	target := "https://y.gtimg.cn/music/photo_new/" + prefix + "R" + itoa(size) + "x" + itoa(size) + "M000" + id + ".jpg?max_age=2592000"
	return routeResult{status: 302, redirect: target}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
