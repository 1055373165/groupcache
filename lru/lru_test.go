package lru

import (
	"log"
	"testing"
)

type MyType string

func (m MyType) Len() int {
	return len(m)
}

func TestLru(t *testing.T) {
	v1 := MyType("12345")
	v2 := MyType("23456")
	v3 := MyType("34567")
	lru := NewLRUCache(20, nil)
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
	if lru.root.Front().Value.(*Entry).Val != v2 {
		t.Fatalf("expect lru cache queue front value is %s, but got %s", v2, lru.root.Front().Value.(*Entry).Val)
	}
	lru.Put("33333", v3)
	// 淘汰掉 key = 11111
	if lru.Len() != expect {
		t.Fatalf("expect lru length is %d but got %d", expect, lru.Len())
	}
	// 队头元素应该是 34567
	if lru.root.Front().Value.(*Entry).Val != v3 {
		t.Fatalf("expect lru cache queue front value is %s, but got %s", v3, lru.root.Front().Value.(*Entry).Val)
	}
	// 查询不到 key = 11111
	if _, ok := lru.Get("11111"); ok {
		t.Fatal("key should be die out")
	}
}
