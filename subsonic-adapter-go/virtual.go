package main

import "strings"

// 虚拟歌单 id 编解码，对应 Node 版 virtual.ts。
// 格式：virtual:<kind>[:<arg>]

const virtualPrefix = "virtual:"

var virtualKinds = map[string]bool{
	"daily": true, "daily30": true, "radar": true, "toplist": true,
	"category": true, "radio": true, "guess": true,
}

type VirtualRef struct {
	Kind string
	Arg  string
}

func virtualID(ref VirtualRef) string {
	if ref.Arg != "" {
		return virtualPrefix + ref.Kind + ":" + ref.Arg
	}
	return virtualPrefix + ref.Kind
}

func parseVirtualID(id string) (VirtualRef, bool) {
	if !strings.HasPrefix(id, virtualPrefix) {
		return VirtualRef{}, false
	}
	body := strings.TrimPrefix(id, virtualPrefix)
	parts := strings.SplitN(body, ":", 2)
	kind := parts[0]
	if !virtualKinds[kind] {
		return VirtualRef{}, false
	}
	ref := VirtualRef{Kind: kind}
	if len(parts) > 1 {
		ref.Arg = parts[1]
	}
	return ref, true
}
