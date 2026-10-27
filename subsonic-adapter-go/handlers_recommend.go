package main

import "regexp"

// 推荐卡片端点。对应 Node 版 library.ts 的 getRecommendations。

var rankCoverPattern = regexp.MustCompile(`T003R\d+x\d+M000([0-9A-Za-z]+)\.`)

func handleGetRecommendations(c routeCtx) routeResult {
	typ := strOf(c.query, "type")
	if typ == "" {
		typ = "daily"
	}
	sections := []map[string]any{}
	switch typ {
	case "daily":
		radarCover, dailyCover := "", ""
		if d := c.upstream.Get("/recommend/radar", nil); d != nil {
			if s := songsOf(dict(d)["songs"]); len(s) > 0 {
				radarCover = s[0].CoverArt
			}
		}
		if d := c.upstream.Get("/user/daily-songs", nil); d != nil {
			if s := songsOf(firstNonNil(dict(d)["songs"], dict(d)["songlist"])); len(s) > 0 {
				dailyCover = s[0].CoverArt
			}
		}
		sections = append(sections, card("daily", "每日推荐", virtualID(VirtualRef{Kind: "daily"}), radarCover))
		sections = append(sections, card("daily30", "每日30首", virtualID(VirtualRef{Kind: "daily30"}), dailyCover))
		for _, o := range []struct{ id, title string }{{"211111", "百万收藏"}, {"211207", "新歌推荐"}, {"211192", "歌手漫游"}} {
			if cd := loadOfficialCard(c, o.id, o.title); cd != nil {
				sections = append(sections, cd)
			}
		}
	case "guess":
		sections = append(sections, card("guess", "猜你喜欢", virtualID(VirtualRef{Kind: "guess"}), ""))
	case "rank":
		data := c.upstream.Get("/getTopLists", nil)
		topList, _ := dict(dict(dict(data)["response"])["data"])["topList"].([]any)
		for _, entry := range topList {
			item := dict(entry)
			id := idOf(firstNonNil(item["id"], item["topId"]))
			if id == "" {
				continue
			}
			cover := ""
			if pic, ok := item["picUrl"].(string); ok {
				if m := rankCoverPattern.FindStringSubmatch(pic); len(m) > 1 {
					cover = "top_" + m[1]
				}
			}
			title := "榜单 " + id
			if t, ok := item["topTitle"].(string); ok && t != "" {
				title = t
			}
			sections = append(sections, card("rank-"+id, title, virtualID(VirtualRef{Kind: "toplist", Arg: id}), cover))
		}
	case "radio":
		data := c.upstream.Get("/radio/list", nil)
		list, _ := dict(data)["radios"].([]any)
		for _, entry := range list {
			item := dict(entry)
			id := idOf(firstNonNil(item["id"], item["radioId"]))
			if id == "" {
				continue
			}
			title := "电台 " + id
			if t, ok := item["name"].(string); ok && t != "" {
				title = t
			}
			sections = append(sections, card("radio-"+id, title, virtualID(VirtualRef{Kind: "radio", Arg: id}), ""))
		}
	case "categories":
		data := c.upstream.Get("/getSongListCategories", nil)
		groups, _ := dict(dict(dict(data)["response"])["data"])["categories"].([]any)
		for _, group := range groups {
			items, _ := dict(group)["items"].([]any)
			for _, entry := range items {
				item := dict(entry)
				id := idOf(firstNonNil(item["categoryId"], item["id"]))
				if id == "" {
					continue
				}
				title := "分类 " + id
				if t, ok := item["categoryName"].(string); ok && t != "" {
					title = t
				}
				sections = append(sections, card("cat-"+id, title, virtualID(VirtualRef{Kind: "category", Arg: id}), ""))
			}
		}
	}
	return c.ok(map[string]any{"recommendations": map[string]any{"source": "upstream", "type": typ, "sections": sections}})
}
