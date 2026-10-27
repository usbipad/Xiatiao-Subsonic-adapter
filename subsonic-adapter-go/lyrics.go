package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// 歌词解析，对应 Node 版 lyrics.ts。

type LyricLine struct {
	Value string `json:"value"`
	Start int    `json:"start"`
}

type StructuredLyrics struct {
	Lang   string      `json:"lang,omitempty"`
	Synced bool        `json:"synced"`
	Line   []LyricLine `json:"line"`
}

var timeTag = regexp.MustCompile(`\[(\d{1,3}):(\d{1,2})(?:[.:](\d{1,3}))?\]`)
var metaTag = regexp.MustCompile(`(?i)^\[(ti|ar|al|by|offset|length|re|ve):.*\]$`)

func toMs(minutes, seconds, fraction string) int {
	mm := 0
	ss := 0
	fmt.Sscanf(minutes, "%d", &mm)
	fmt.Sscanf(seconds, "%d", &ss)
	ms := 0
	if fraction != "" {
		v := 0
		fmt.Sscanf(fraction, "%d", &v)
		switch len(fraction) {
		case 1:
			ms = v * 100
		case 2:
			ms = v * 10
		default:
			ms = v
		}
	}
	return mm*60000 + ss*1000 + ms
}

func parseLRC(raw string) []LyricLine {
	if raw == "" {
		return nil
	}
	type tmp struct {
		start int
		value string
	}
	out := []tmp{}
	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(rawLine, "\r"))
		if line == "" || metaTag.MatchString(line) {
			continue
		}
		stamps := []int{}
		for _, m := range timeTag.FindAllStringSubmatch(line, -1) {
			stamps = append(stamps, toMs(m[1], m[2], m[3]))
		}
		value := strings.TrimSpace(timeTag.ReplaceAllString(line, ""))
		if value == "" {
			continue
		}
		if len(stamps) == 0 {
			out = append(out, tmp{0, value})
			continue
		}
		for _, s := range stamps {
			out = append(out, tmp{s, value})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].start < out[j].start })
	res := make([]LyricLine, 0, len(out))
	for _, t := range out {
		res = append(res, LyricLine{Value: t.value, Start: t.start})
	}
	return res
}

func plainLyrics(raw string) []LyricLine {
	res := []LyricLine{}
	for _, line := range strings.Split(raw, "\n") {
		if v := strings.TrimSpace(strings.TrimSuffix(line, "\r")); v != "" {
			res = append(res, LyricLine{Value: v, Start: 0})
		}
	}
	return res
}

func structuredLyricsOf(raw, lang string) []StructuredLyrics {
	if raw == "" {
		return nil
	}
	hasTimeline := regexp.MustCompile(`\[\d{1,3}:\d{1,2}`).MatchString(raw)
	var lines []LyricLine
	if hasTimeline {
		lines = parseLRC(raw)
	} else {
		lines = plainLyrics(raw)
	}
	if len(lines) == 0 {
		return nil
	}
	return []StructuredLyrics{{Lang: lang, Synced: hasTimeline, Line: lines}}
}
