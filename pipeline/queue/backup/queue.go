package queue

import (
	"container/list"
	"context"
	"sync"
)

// Item represents an item in the Queue.
type Item struct {
	ID string
}

// Queue represents a collection of Items pending processing.
type Queue interface {
	Enqueue(context.Context, *Item)
	Dequeue(context.Context) *Item
	Remove(context.Context, string)
	Slice(context.Context) []*Item
}

type queue struct {
	sync.RWMutex

	subs map[interface{}]struct{}
	list *list.List
}

func (q *queue) Enqueue(c context.Context, item *Item) {
	q.Lock()
	q.list.PushBack(item)
	q.Unlock()
}

func (q *queue) Dequeue(c context.Context) *Item {
	return nil
}

func (q *queue) Remove(c context.Context, id string) {
	var next *list.Element

	q.Lock()
	for e := q.list.Front(); e != nil; e = next {
		item := e.Value.(*Item)
		if item.ID == id {
			q.list.Remove(e)
			break
		}
	}
	q.Unlock()
}

func (q *queue) Size(context.Context) int {
	q.RLock()
	size := q.list.Len()
	q.RUnlock()
	return size
}

func (q *queue) Slice(context.Context) []*Item {
	var slice []*Item
	var next *list.Element

	q.RLock()
	for e := q.list.Front(); e != nil; e = next {
		slice = append(slice, e.Value.(*Item))
	}
	q.RUnlock()
	return slice
}
