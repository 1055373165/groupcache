package policy

import (
	"log"
	"reflect"
	"testing"

	"github.com/1055373165/groupcache/conf"
)

type String string

func (s String) Len() int {
	return len(s)
}

func TestGet(t *testing.T) {
	lru := NewLRUCache(15, nil)
	lru.Put("key1", String("1234"))
	if v, _, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	curcap := len(k1 + k2 + v1 + v2)
	lru := NewLRUCache(int64(curcap), nil)
	lru.Put(k1, String(v1))
	lru.Put(k2, String(v2))
	lru.Put(k3, String(v3))
	if _, _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("removeoldest key1 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := NewLRUCache(10, callback)
	lru.Put("key1", String("123456"))
	// 删除 key1 value1 将 key1 保存起来
	lru.Put("k2", String("k2"))
	lru.Put("k3", String("k3"))
	// 删除 key2 value2 将 key2 保存起来
	lru.Put("k4", String("k4"))

	expect := []string{"key1", "k2"}
	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("call OnEnvicted failed, expect keys equals to %v but got %v", expect, keys)
	}
}

type MyType string

func (m MyType) Len() int {
	return len(m)
}

func init() {
	conf.Init()
}

func TestLru(t *testing.T) {
	v1 := MyType("12345")
	v2 := MyType("23456")
	v3 := MyType("34567")
	lru := NewLRUCache(20, nil)
	if lru == nil {
		t.Fatal("lru is nil")
	}
	log.Println(lru.Len())
	log.Println("1--------")
	lru.Put("11111", v1)
	lru.Put("22222", v2)
	log.Println(lru.Len())
	expect := 2
	log.Println("2--------")
	if lru.Len() != expect {
		t.Fatalf("expect lru length is %d but got %d", expect, lru.Len())
	}
	log.Println("3--------")
	// 队头元素应该是 23456
	if lru.root.Front().Value.(*entry).value != v2 {
		t.Fatalf("expect lru cache queue front value is %s, but got %s", v2, lru.root.Front().Value.(*entry).value)
	}
	lru.Put("33333", v3)
	// 淘汰掉 key = 11111
	if lru.Len() != expect {
		t.Fatalf("expect lru length is %d but got %d", expect, lru.Len())
	}
	// 队头元素应该是 34567
	if lru.root.Front().Value.(*entry).value != v3 {
		t.Fatalf("expect lru cache queue front value is %s, but got %s", v3, lru.root.Front().Value.(*entry).value)
	}
	// 查询不到 key = 11111
	if _, _, ok := lru.Get("11111"); ok {
		t.Fatal("key should be die out")
	}
}
