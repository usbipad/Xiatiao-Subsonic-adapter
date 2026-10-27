package main

import (
	"regexp"
	"strings"
)

// 歌单与虚拟歌单端点。对应 Node 版 library.ts。

var coverIDPattern = regexp.MustCompile(`M000([0-9A-Za-z]+)\.`)

func coverIDOf(u string) string {
	if u == "" {
		return ""
	}
	if m := coverIDPattern.FindStringSubmatch(u); len(m) > 1 {
		return m[1]
	}
	return ""
}

func firstString(values ...any) string {
	for _, v := range values {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func handleGetPlaylists(c routeCtx) routeResult {
	data := c.upstream.Get("/user/playlist", nil)
	list, _ := dict(data)["playlist"].([]any)
	playlists := []map[string]any{}
	for _, entry := range list {
		p := dict(entry)
		id := idOf(firstNonNil(p["dissid"], p["tid"], p["dirId"], p["dirid"], p["id"]))
		if id == "" {
			continue
		}
		name := firstString(p["dissname"], p["dirName"], p["name"], p["title"])
		if name == "" {
			name = id
		}
		item := map[string]any{"id": id, "name": name}
		if n, ok := num(p["songnum"]); ok {
			item["songCount"] = n
		}
		cover := coverIDOf(firstString(p["bigpicUrl"], p["picUrl"], p["logo"], p["albumPicUrl"]))
		if cover != "" {
			item["coverArt"] = cover
		}
		playlists = append(playlists, item)
	}
	for _, p := range playlists {
		if _, has := p["coverArt"]; has {
			continue
		}
		liked := c.upstream.Get("/user/liked-songs", map[string]string{"offset": "0", "limit": "1"})
		songs := songsOf(dict(liked)["songs"])
		if len(songs) > 0 && songs[0].CoverArt != "" {
			p["coverArt"] = songs[0].CoverArt
		}
		break
	}
	return c.ok(map[string]any{"playlists": map[string]any{"playlist": playlists}})
}

func handleGetPlaylist(c routeCtx) routeResult {
	id := strOf(c.query, "id")
	if id == "" {
		return c.fail(ErrMissingParameter, "id required")
	}
	if ref, ok := parseVirtualID(id); ok {
		return virtualPlaylist(c, ref)
	}
	data := c.upstream.Get("/getSongListDetail", map[string]string{"disstid": id})
	response := dict(dict(data)["response"])
	cdlist, _ := response["cdlist"].([]any)
	var cd map[string]any
	if len(cdlist) > 0 {
		cd = dict(cdlist[0])
	}
	entries, _ := cd["songlist"].([]any)
	songs := songsOf(entries)
	name := id
	if s, ok := cd["dissname"].(string); ok && s != "" {
		name = s
	}
	return c.ok(map[string]any{"playlist": playlistJSON(id, name, len(songs), "", songs)})
}

func containsSong(songs []Song, id string) bool {
	for _, s := range songs {
		if s.ID == id {
			return true
		}
	}
	return false
}
