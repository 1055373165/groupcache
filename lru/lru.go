package lru

import (
	"container/list"
	"etcd/logger"
)

type LRUCache struct {
	maxCacheSize int64
	nBytes       int64
	root         *list.List
	m            map[string]*list.Element
	onEcvited    func(string, Value)
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

type Entry struct {
	Key string
	Val Value
}

func NewEntry(key string, val Value) *Entry {
	return &Entry{
		Key: key,
		Val: val,
	}
}

func (l *LRUCache) Get(key string) (Value, bool) {
	logger.Logger.Info("lru.Get()")
	if e, ok := l.m[key]; ok {
		l.root.MoveToFront(e)
		kv := e.Value.(*Entry)
		logger.Logger.Info("lru.Get 断言")
		return kv.Val, true
	} else {
		return nil, false
	}
}

func (l *LRUCache) Put(key string, value Value) {
	logger.Logger.Info("lru.Put(key, val)")
	if e, ok := l.m[key]; ok {
		l.root.MoveToFront(e)
		logger.Logger.Info("lru.*Entry断言")
		l.nBytes += int64(value.Len()) - int64(e.Value.(*Entry).Val.Len())
		e.Value = value
	} else {
		newEntry := NewEntry(key, value)
		ele := l.root.PushFront(newEntry)
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
		kv := kvInterface.(*Entry)
		l.nBytes -= int64(kv.Val.Len()) + int64(len(kv.Key))
		delete(l.m, kv.Key)
		if l.onEcvited != nil {
			l.onEcvited(kv.Key, kv.Val)
		}
	}
}

func (l *LRUCache) Len() int {
	return l.root.Len()
}
