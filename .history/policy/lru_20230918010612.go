package policy

import (
	"container/list"
	"time"

	"github.com/1055373165/groupcache/logger"
)

type LRUCache struct {
	maxCacheSize int64
	nBytes       int64
	root         *list.List
	m            map[string]*list.Element
	// 可选并在清除条目时执行。
	// 回调函数，采用依赖注入的方式，该函数用于处理从缓存中淘汰的数据
	onEcvited func(string, Value)
}

type Value interface {
	Len() int
}

func NewLRUCache(capacity int64, onEvicted func(string, Value)) *LRUCache {
	return &LRUCache{
		maxCacheSize: capacity,
		root:         list.New(),
		m:            make(map[string]*list.Element),
		onEcvited:    onEvicted,
	}
}

func (l *LRUCache) Get(key string) (Value, *time.Time, bool) {
	logger.Logger.Info("lru.Get()")
	if ele, ok := l.m[key]; ok {
		l.root.MoveToFront(ele)
		e := ele.Value.(*entry)
		// 频次和更新时间
		e.touch()
		return e.value, e.updateAt, ok
	} else {
		return nil, nil, false
	}
}

func (l *LRUCache) Put(key string, value Value) {
	logger.Logger.Info("lru.Put(key, val)")
	if e, ok := l.m[key]; ok {
		l.root.MoveToFront(e)
		kv := e.Value.(*entry)
		l.nBytes += int64(value.Len()) - int64(kv.value.Len())
		e.Value = value
	} else {
		kv := &entry{key, value, nil}
		kv.touch()
		ele := l.root.PushFront(kv)
		l.nBytes += int64(value.Len()) + int64(len(key))
		l.m[key] = ele
	}

	if l.maxCacheSize != 0 && l.maxCacheSize < l.nBytes {
		l.RemoveOldest()
	}
}

func (l *LRUCache) RemoveOldest() {
	logger.Logger.Info("lru.RemoveOldest(key, val)")
	for l.maxCacheSize < l.nBytes {
		kvInterface := l.root.Remove(l.root.Back())
		logger.Logger.Info("lru.removeOldest()")
		kv := kvInterface.(*entry)
		l.nBytes -= int64(kv.value.Len()) + int64(len(kv.key))
		delete(l.m, kv.key)
		if l.onEcvited != nil {
			l.onEcvited(kv.key, kv.value)
		}
	}
}

func (l *LRUCache) CleanUp(ttl time.Duration) {
	for e := l.root.Front(); e != nil; e = e.Next() {
		if e.Value.(*entry).expired(ttl) {
			kv := l.root.Remove(e).(*entry)
			delete(l.m, kv.key)
			l.nBytes -= int64(len(kv.key) + kv.value.Len())
			if l.onEcvited != nil {
				l.onEcvited(kv.key, kv.value)
			}
		} else {
			break
		}
	}
}

func (l *LRUCache) Len() int {
	return l.root.Len()
}
