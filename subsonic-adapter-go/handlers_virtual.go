package main

import (
	"strings"
)

// 虚拟歌单展开与推荐卡片。对应 Node 版 library.ts 的 virtualPlaylist / getRecommendations。

func excludeSet(c routeCtx) map[string]bool {
	ex := map[string]bool{}
	for _, v := range strings.Split(strOf(c.query, "exclude"), ",") {
		if t := strings.TrimSpace(v); t != "" {
			ex[t] = true
		}
	}
	return ex
}

func fetchRadar(c routeCtx) []Song {
	data := c.upstream.Get("/recommend/radar", nil)
	return songsOf(dict(data)["songs"])
}

func virtualPlaylist(c routeCtx, ref VirtualRef) routeResult {
	switch ref.Kind {
	case "daily30":
		data := c.upstream.Get("/user/daily-songs", nil)
		songs := songsOf(firstNonNil(dict(data)["songs"], dict(data)["songlist"]))
		cover := ""
		if len(songs) > 0 {
			cover = songs[0].CoverArt
		}
		return c.ok(map[string]any{"playlist": playlistJSON(virtualID(ref), "每日30首", len(songs), cover, songs)})
	case "radar", "daily":
		ex := excludeSet(c)
		filtered := []Song{}
		for _, s := range fetchRadar(c) {
			if !ex[s.ID] {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) < 5 {
			for _, s := range fetchRadar(c) {
				if ex[s.ID] || containsSong(filtered, s.ID) {
					continue
				}
				filtered = append(filtered, s)
			}
		}
		name := "每日推荐"
		if ref.Kind == "radar" {
			name = "雷达模式"
		}
		cover := ""
		if len(filtered) > 0 {
			cover = filtered[0].CoverArt
		}
		return c.ok(map[string]any{"playlist": playlistJSON(virtualID(ref), name, len(filtered), cover, filtered)})
	case "guess":
		data := c.upstream.Get("/user/guess-you-like", nil)
		songs := songsOf(firstNonNil(dict(data)["songs"], dict(data)["songlist"]))
		cover := ""
		if len(songs) > 0 {
			cover = songs[0].CoverArt
		}
		return c.ok(map[string]any{"playlist": playlistJSON(virtualID(ref), "猜你喜欢", len(songs), cover, songs)})
	case "toplist":
		return toplistPlaylist(c, ref)
	case "radio":
		if ref.Arg == "" {
			return c.fail(ErrNotFound, "not found")
		}
		data := c.upstream.Get("/radio/tracks", map[string]string{"id": ref.Arg, "limit": "100"})
		songs := songsOf(firstNonNil(dict(data)["songs"], dict(data)["songlist"]))
		return c.ok(map[string]any{"playlist": playlistJSON(virtualID(ref), "电台 "+ref.Arg, len(songs), "", songs)})
	case "category":
		if ref.Arg == "" {
			return c.fail(ErrNotFound, "not found")
		}
		data := c.upstream.Get("/getSongLists", map[string]string{"page": "1", "limit": "20", "categoryId": ref.Arg, "sortId": "5"})
		list, _ := dict(dict(dict(data)["response"])["data"])["list"].([]any)
		var first map[string]any
		if len(list) > 0 {
			first = dict(list[0])
		}
		songs := songsOf(first["songlist"])
		name := "分类 " + ref.Arg
		if s, ok := first["dissname"].(string); ok && s != "" {
			name = s
		}
		return c.ok(map[string]any{"playlist": playlistJSON(virtualID(ref), name, len(songs), "", songs)})
	}
	return c.fail(ErrNotFound, "not found")
}

func toplistPlaylist(c routeCtx, ref VirtualRef) routeResult {
	if ref.Arg == "" {
		return c.fail(ErrNotFound, "not found")
	}
	data := c.upstream.Get("/getRanks", map[string]string{"topId": ref.Arg, "limit": "100", "page": "1"})
	inner := dict(dict(dict(dict(dict(data)["response"])["req_1"])["data"])["data"])
	list, _ := inner["song"].([]any)
	songs := []Song{}
	for _, entry := range list {
		item := dict(entry)
		songID := idOf(firstNonNil(item["songId"], item["mid"], item["songmid"]))
		if songID == "" {
			continue
		}
		s := Song{ID: songID, Title: songID}
		if t, ok := item["title"].(string); ok && t != "" {
			s.Title = t
		}
		if a, ok := item["singerName"].(string); ok {
			s.Artist = a
		}
		if am, ok := item["albumMid"].(string); ok && am != "" {
			s.CoverArt = am
			s.Album = am
		}
		if numericIDPattern.MatchString(songID) {
			rememberSong(songID, songHint{Title: s.Title, Artist: s.Artist})
		}
		songs = append(songs, s)
	}
	title := "榜单 " + ref.Arg
	if t, ok := inner["title"].(string); ok && t != "" {
		title = t
	}
	cover := ""
	if len(songs) > 0 {
		cover = songs[0].CoverArt
	}
	return c.ok(map[string]any{"playlist": playlistJSON(virtualID(ref), title, len(songs), cover, songs)})
}

func card(id, title, playlistID, coverArt string) map[string]any {
	m := map[string]any{"id": id, "title": title, "kind": "playlist", "playlistId": playlistID}
	if coverArt != "" {
		m["coverArt"] = coverArt
	}
	return m
}

func loadOfficialCard(c routeCtx, disstid, fallbackTitle string) map[string]any {
	data := c.upstream.Get("/getSongListDetail", map[string]string{"disstid": disstid})
	if data == nil {
		return nil
	}
	response := dict(dict(data)["response"])
	cdlist, _ := response["cdlist"].([]any)
	if len(cdlist) == 0 {
		return nil
	}
	cd := dict(cdlist[0])
	songs := songsOf(cd["songlist"])
	if len(songs) == 0 {
		return nil
	}
	title := fallbackTitle
	if s, ok := cd["dissname"].(string); ok && s != "" {
		title = s
	}
	m := map[string]any{"id": "official-" + disstid, "title": title, "kind": "playlist", "playlistId": disstid}
	if songs[0].CoverArt != "" {
		m["coverArt"] = songs[0].CoverArt
	}
	return m
}
