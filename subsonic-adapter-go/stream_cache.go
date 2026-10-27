package main

import (
	"sync"
	"time"
)

// 直链缓存：按 songmid+quality 缓存上游直链，避免每次 seek 都重新换链。
// 上游 getMusicPlay 对频繁调用有限制，缓存是 seek 快慢的关键。

const (
	streamCacheTTL        = 5 * time.Minute
	streamCacheMaxEntries = 2048
)

type cacheEntry struct {
	url     string
	expires time.Time
}

type streamCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

func newStreamCache() *streamCache {
	return &streamCache{entries: map[string]cacheEntry{}}
}

func (c *streamCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ent, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if time.Now().After(ent.expires) {
		delete(c.entries, key)
		return "", false
	}
	return ent.url, true
}

func (c *streamCache) set(key, url string) {
	if url == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= streamCacheMaxEntries {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expires) {
				delete(c.entries, k)
			}
		}
	}
	c.entries[key] = cacheEntry{url: url, expires: time.Now().Add(streamCacheTTL)}
}

func (c *streamCache) invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// 全局缓存实例。
var globalStreamCache = newStreamCache()
