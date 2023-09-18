package policy

import "testing"

func TestPriorityQueue_Get(t *testing.T) {
	lfu := NewLfuCache(10, nil)
	// 8
	lfu.Put("key1", String("1234"))
	if v, _, ok := lfu.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	// 超出缓存，按照频次，key2 被移除
	lfu.Put("key2", String("1234"))
	if _, _, ok := lfu.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestPriorityQueue_Remove(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	curCap := len(k1 + k2 + v1 + v2)
	lfu := NewLfuCache(int64(curCap), nil)
	lfu.Put(k1, String(v1))
	lfu.Put(k1, String(v1))
	// 加入后刚好等于容量上限，需要移除一个频次较小的，那就是 key2
	lfu.Put(k2, String(v2))
	// key3 加入后不超过容量上限保留
	lfu.Put(k3, String(v3))

	//for k, v := range lfu.cache {
	//	fmt.Printf("%s%v\n", k, v)
	//}
	if _, _, ok := lfu.Get("key2"); ok || lfu.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}

	if _, _, ok := lfu.Get("key3"); ok || lfu.Len() != 2 {
		t.Fatalf("key3 is not store but can got")
	}

	lfu.Put(k3, String(v3))
	if _, _, ok := lfu.Get("k3"); !ok || lfu.Len() != 2 {
		t.Fatalf("key3 is store but can't got")
	}
}
