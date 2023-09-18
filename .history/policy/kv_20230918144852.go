package policy

import "time"

type Cache interface {
	Get(string) (Value, *time.Time, bool)
	Put(string, Value)
	CleanUp(time.Duration)
	Len() int
}

type entry struct {
	key      string
	value    Value
	updateAt *time.Time
}

// TTL
func (e *entry) expired(duration time.Duration) (ok bool) {
	if e.updateAt == nil {
		ok = false
	} else {
		ok = e.updateAt.Add(duration).Before(time.Now())
	}
	return
}

// 访问时间更新
func (e *entry) touch() {
	nowTime := time.Now()
	e.updateAt = &nowTime
}

func New(name string, maxBytes int64, onEvicted func(string, Value)) Cache {
	switch name {
	case "lru":
		return NewLRUCache(maxBytes, onEvicted)
	case "lfu":
		return NewLfuCache(maxBytes, onEvicted)
	default:
	}

	return nil
}


