package policy

// 手撕优先队列（未使用 golang 自带 heap 容器）
// 实现 container/heap
type priorityqueue []*LfuEntry

// 实现 container/sort.Interface
func (pq priorityqueue) Len() int {
	return len(pq)
}

// 两个维度：1、访问频次 2、过期时间（过期时间靠后的是最近刚访问过的）
func (pq priorityqueue) Less(i, j int) bool {
	// 在两个条目访问频次相同的情况下，取过期时间较早的那个
	if pq[i].count == pq[j].count {
		return pq[i].entry.updateAt.Before(*pq[j].entry.updateAt)
	}

	return pq[i].count < pq[j].count
}

// 按照访问频次从小到大排序，访问越多的条目越靠后
// 在访问频次相同时，按照最近访问时间进行排序，过期时间较晚的排在后面（更优先）
func (pq priorityqueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityqueue) Push(x interface{}) {
	entry := x.(*LfuEntry)
	entry.index = len(*pq) // 在数组中的下标
	*pq = append(*pq, x.(*LfuEntry))
}

func (pq *priorityqueue) Pop() interface{} {
	oldpq := *pq
	n := len(oldpq)
	entry := oldpq[n-1]
	oldpq[n-1] = nil // 避免内存泄露
	newpq := oldpq[:n-1]
	// 重排索引（因为堆会触发向上向下调整）
	for i := 0; i < len(newpq); i++ {
		newpq[i].index = i
	}

	*pq = newpq
	return entry
}
