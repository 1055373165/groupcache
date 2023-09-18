package policy

import (
	"container/heap"
	"fmt"
	"testing"
)

func TestPriority(t *testing.T) {
	pq := priorityqueue([]*LfuEntry{})
	// heap.Init(&pq) 如果没有初始元素可以省略这个步骤
	for i := 0; i < 10; i++ {
		heap.Push(&pq, &LfuEntry{i, entry{}, i})
	}

	for pq.Len() != 0 {
		e := heap.Pop(&pq).(*LfuEntry)

		fmt.Println("count:", e.count)
		if pq.Len() > 0 {
			fmt.Println("pq.index:", pq[pq.Len()-1].index)
		}
	}
}
