package group

import (
	"sync"

	"github.com/1055373165/groupcache/logger"
	"github.com/1055373165/groupcache/policy"
)

// cache 模块负责提供对lru模块的并发控制

// 给 lru 上层并发上一层锁
type cache struct {
	mu           sync.Mutex
	lru          *policy.LRUCache
	maxCacheSize int64 // 保证 lru 一定初始化
}

func newCache(cacheSize int64) *cache {
	return &cache{
		maxCacheSize: cacheSize,
	}
}

// 并发控制
func (c *cache) set(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = policy.NewLRUCache(c.maxCacheSize, nil)
	}
	c.lru.Put(key, value)
}

func (c *cache) get(key string) (ByteView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = policy.NewLRUCache(c.maxCacheSize, nil)
	}

	if v, _, ok := c.lru.Get(key); ok { // Get 返回值是 Value 接口，直接类型断言
		return v.(ByteView), true
	} else {
		return ByteView{}, false
	}
}

func (c *cache) put(key string, val ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = policy.NewLRUCache(c.maxCacheSize, nil)
	}
	logger.Logger.Info("cache.put(key, val)")
	c.lru.Put(key, val)
}
