package main

import (
	"fmt"
	"regexp"
	"strings"
)

// 上游响应 → Subsonic 领域对象映射，对应 Node 版 mapper.ts。

type QualityLevel struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Br   int    `json:"br"`
}

type Song struct {
	ID            string
	Title         string
	Artist        string
	Album         string
	Duration      *int
	CoverArt      string
	Suffix        string
	BitRate       *int
	QualityLevels []QualityLevel
}

func dict(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func read(v any, key string) any { return dict(v)[key] }

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func num(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case string:
		var out int
		if _, err := fmt.Sscanf(n, "%d", &out); err == nil {
			return out, true
		}
	}
	return 0, false
}

func idOf(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%d", int64(x))
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	}
	return ""
}

func firstText(values ...any) string {
	for _, v := range values {
		if t := strings.TrimSpace(str(v)); t != "" {
			return t
		}
	}
	return ""
}

var albumMidPattern = regexp.MustCompile(`^[0-9A-Za-z]+$`)

func artistNameOf(song any) string {
	singers, ok := read(song, "singer").([]any)
	if ok {
		names := []string{}
		for _, s := range singers {
			if n := firstText(read(s, "name"), read(s, "title")); n != "" {
				names = append(names, n)
			}
		}
		if len(names) > 0 {
			return strings.Join(names, " / ")
		}
	}
	return firstText(read(song, "artist"), read(song, "singername"))
}

type albumInfo struct {
	Name string
	Mid  string
}

func albumOf(song any) albumInfo {
	album := read(song, "album")
	if m, ok := album.(map[string]any); ok {
		return albumInfo{
			Name: firstText(read(m, "name"), read(m, "title")),
			Mid:  firstText(read(m, "mid")),
		}
	}
	return albumInfo{Name: firstText(read(song, "albumname"))}
}

func fileSuffixOf(file map[string]any) string {
	type cand struct {
		key string
		suf string
	}
	for _, c := range []cand{
		{"size_flac", "flac"}, {"size_320mp3", "mp3"}, {"size_128mp3", "mp3"},
		{"size_m4a", "m4a"}, {"size_ape", "ape"},
	} {
		if n, ok := num(file[c.key]); ok && n > 0 {
			return c.suf
		}
	}
	return ""
}

func qualityLevelsOf(file map[string]any) []QualityLevel {
	levels := []QualityLevel{}
	if n, ok := num(file["size_128mp3"]); ok && n > 0 {
		levels = append(levels, QualityLevel{Key: "standard", Name: "标准", Br: 128})
	}
	if n, ok := num(file["size_320mp3"]); ok && n > 0 {
		levels = append(levels, QualityLevel{Key: "high", Name: "高品", Br: 320})
	}
	flacN, flacOK := num(file["size_flac"])
	apeN, apeOK := num(file["size_ape"])
	if (flacOK && flacN > 0) || (apeOK && apeN > 0) {
		levels = append(levels, QualityLevel{Key: "lossless", Name: "无损", Br: 999})
	}
	return levels
}

// songOf 把上游条目解析为 Song；无 id 返回 nil。
func songOf(entry any) *Song {
	song := dict(entry)
	id := idOf(firstNonNil(song["mid"], song["songmid"], song["id"], song["songid"]))
	if id == "" {
		return nil
	}
	album := albumOf(entry)
	title := firstText(read(entry, "songname"), read(entry, "name"), read(entry, "title"))
	if title == "" {
		title = id
	}
	artist := artistNameOf(entry)
	var duration *int
	if d, ok := num(firstNonNil(read(entry, "interval"), read(entry, "duration"))); ok && d >= 0 {
		duration = &d
	}
	file := dict(read(entry, "file"))
	suffix := fileSuffixOf(file)
	levels := qualityLevelsOf(file)
	out := &Song{ID: id, Title: title, Artist: artist, Duration: duration, Suffix: suffix, QualityLevels: levels}
	if album.Name != "" {
		out.Album = album.Name
	}
	if album.Mid != "" && albumMidPattern.MatchString(album.Mid) {
		out.CoverArt = album.Mid
	}
	return out
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func songsOf(entries any) []Song {
	arr, ok := entries.([]any)
	if !ok {
		return nil
	}
	out := []Song{}
	for _, e := range arr {
		if s := songOf(e); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// songJSON 输出 Subsonic 歌曲 JSON。
func songJSON(s Song) map[string]any {
	m := map[string]any{"id": s.ID, "title": s.Title}
	if s.Artist != "" {
		m["artist"] = s.Artist
	}
	if s.Album != "" {
		m["album"] = s.Album
	}
	if s.Duration != nil {
		m["duration"] = *s.Duration
	}
	if s.CoverArt != "" {
		m["coverArt"] = s.CoverArt
	}
	if s.Suffix != "" {
		m["suffix"] = s.Suffix
	}
	if s.BitRate != nil {
		m["bitRate"] = *s.BitRate
	}
	if len(s.QualityLevels) > 0 {
		m["qualityLevels"] = s.QualityLevels
	}
	return m
}

func songsJSON(songs []Song) []map[string]any {
	out := make([]map[string]any, 0, len(songs))
	for _, s := range songs {
		out = append(out, songJSON(s))
	}
	return out
}

// playlistJSON 输出 Subsonic 歌单 JSON。
func playlistJSON(id, name string, songCount int, coverArt string, entries []Song) map[string]any {
	m := map[string]any{"id": id, "name": name, "songCount": songCount}
	if coverArt != "" {
		m["coverArt"] = coverArt
	}
	if len(entries) > 0 {
		m["entry"] = songsJSON(entries)
	}
	return m
}
