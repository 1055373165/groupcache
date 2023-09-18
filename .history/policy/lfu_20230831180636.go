package policy

import (
	"container/heap"
	"time"
)

type LfuCache struct {
	nbytes    int64
	maxBytes  int64
	cache     map[string]*LfuEntry
	pq        *priorityqueue // 优先队列
	OnEvicted func(key string, value Value)
}

type LfuEntry struct {
	index int
	entry entry
	count int
}

// 访问频次
// 1. 去缓存查询 key 的值
// 2. 如果可以查询的得到，那么更新访问计数和过期时间
// 3. 根据新的频次调整堆的结构，重新建立堆
// 4. 返回缓存值、最新访问时间以及是否查询的标志
func (lc *LfuCache) Get(key string) (Value, *time.Time, bool) {
	if e, ok := lc.cache[key]; ok {
		e.referenced()
		// 在索引i处的元素更改其值后，FIX重新建立堆排序。
		heap.Fix(lc.pq, e.index)
		return e.entry.value, e.entry.updateAt, ok
	}
	return nil, nil, false
}

/*
1. 先去缓存中查询

2. 如果可以查询得到，那么需要更新当前缓存值（当前使用容量加上用新值和旧值长度的差值）
3. 更新值、更新访问计数和最新访问时间
4. 调整堆（调用 Fix 函数对条目所在的索引进行堆调整）

2. 如果查询不到，那么相当于插入一条新的 kv 对，构造一个 LfuEntry 类型的记录
3. 更新它的访问频次和上一次的更新时间；
4. 插入到堆中并更新缓存容量
5. 将 key 和对应的 LfuEntry 存入缓存

无论最开始是否从缓存中查询到 key 的值，都需要判断当前使用的缓存容量是否超过了缓存上限，如果是
则使用 LFU 缓存淘汰算法淘汰掉一些条目，直至满足缓存上限为止；因为 LFU 使用了优先队列作为数据结构，
实际上底层就是一个小根堆，按照条目的访问频次进行堆排序，相同条目按照更新时间先后进行堆排序，默认最新
更新的条目优先级更高，即更晚从堆中删除
*/
func (lc *LfuCache) Put(key string, value Value) {
	if e, ok := lc.cache[key]; ok {
		// 更新 value
		lc.nbytes += int64(value.Len()) - int64(e.entry.value.Len())
		e.entry.value = value
		e.referenced()
		heap.Fix(lc.pq, e.index) // 从插入的下标处重建堆
	} else {
		e := &LfuEntry{
			index: 0,
			entry: entry{
				key:      key,
				value:    value,
				updateAt: nil,
			},
		}

		e.referenced()
		heap.Push(lc.pq, e)
		lc.cache[key] = e
		lc.nbytes += int64(len(key)) + int64(value.Len())
	}

	for lc.maxBytes != 0 && lc.maxBytes < lc.nbytes {
		lc.Remove()
	}
}

func (lf *LfuCache) Remove() {
	e := heap.Pop(lf.pq).(*LfuEntry)
	delete(lf.cache, e.entry.key)
	lf.nbytes -= int64(len(e.entry.key) + e.entry.value.Len())
	if lf.OnEvicted != nil {
		lf.OnEvicted(e.entry.key, e.entry.value)
	}
}

func (lf *LfuCache) CleanUp(ttl time.Duration) {
	for _, e := range *lf.pq {
		if e.entry.expired(ttl) {
			kv := heap.Remove(lf.pq, e.index).(*LfuEntry).entry
			delete(lf.cache, kv.key)
			lf.nbytes -= int64(len(kv.key) + kv.value.Len())
			if lf.OnEvicted != nil {
				lf.OnEvicted(kv.key, kv.value)
			}
		}
	}
}

func (lf *LfuCache) Len() int {
	return lf.pq.Len()
}

func (lf *LfuEntry) referenced() {
	// 访问计数+1
	lf.count++
	// 更新过期时间
	lf.entry.touch()
}
