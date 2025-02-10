package netproof

import (
	"container/heap"
	"context"
	"sync"
)

type Priority struct {
	items     queue
	itemReady *sync.Cond
	ctx       context.Context
}

// NewPriority creates a new Priority queue.
// When ctx is canceled, it stops any ongoing Pop().
func NewPriority(ctx context.Context) *Priority {
	itemReady := sync.NewCond(&sync.Mutex{})
	go func() {
		<-ctx.Done()
		itemReady.L.Lock()
		itemReady.Broadcast()
		itemReady.L.Unlock()
	}()
	items := make(queue, 0)
	heap.Init(&items)
	return &Priority{ctx: ctx, items: items, itemReady: itemReady}
}

// Push adds an item to the queue (concurrent-safe)
func (p *Priority) Push(prio int, data any) {
	p.itemReady.L.Lock()
	defer p.itemReady.L.Unlock()
	heap.Push(&p.items, &queueItem{prio: prio, data: data})
	p.itemReady.Signal()
}

// Len returns the number of items in the queue (concurrent-safe)
func (p *Priority) Len() int {
	p.itemReady.L.Lock()
	defer p.itemReady.L.Unlock()
	return p.items.Len()
}

// Pop blocks until an item is available (concurrent-safe).
// Removes the item with the highest priority (int value) or the ctx error (if any).
func (p *Priority) Pop() (item any, err error) {
	p.itemReady.L.Lock()
	defer p.itemReady.L.Unlock()
	for {
		if p.ctx.Err() != nil {
			return nil, p.ctx.Err()
		}
		if item, ok := p.next(); ok {
			return item, nil
		}
		p.itemReady.Wait()
	}
}

// next returns the next item (items lock must be held).
func (p *Priority) next() (item any, ok bool) {
	if p.items.Len() == 0 {
		return nil, false
	}
	item = heap.Pop(&p.items).(*queueItem).data
	return item, true
}

type queueItem struct {
	prio int
	data any
}

type queue []*queueItem

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i].prio > q[j].prio }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)        { *q = append(*q, x.(*queueItem)) }
func (q *queue) Pop() any {
	old := *q
	n := len(old)
	x := old[n-1]
	old[n-1] = nil // invalidate the item to allow any GC
	*q = old[0 : n-1]
	return x
}
