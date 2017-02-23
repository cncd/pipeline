package queue

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/cncd/pipeline/pipeline/backend"
)

// Item is an item in the queue.
type Item struct {
	ID      string
	Timeout int64
	Config  *backend.Config
}

// Filter filters items in the queue.
type Filter func(*Item) bool

type queue struct {
	sync.Mutex

	acks map[*Item]time.Time
	subs map[chan *Item]Filter
	list *list.List
}

func newQueue() *queue {
	return &queue{
		acks: map[*Item]time.Time{},
		subs: map[chan *Item]Filter{},
		list: list.New(),
	}
}

// Ack acknowledges the item receipt.
func (q *queue) Ack(c context.Context, item *Item) {
	q.Lock()
	// if _, item := q.acks[item]; ok {
	// 	q.list.PushFront(item)
	// 	delete(q.acks, item)
	// }
	q.Unlock()
	go q.process()
}

// Push pushes an item to the tail of this queue.
func (q *queue) Push(c context.Context, item *Item) {
	q.Lock()
	q.list.PushBack(item)
	q.Unlock()
	go q.process()
}

// Poll retrieves and removes the head of this queue.
func (q *queue) Poll(c context.Context, f Filter) *Item { // consider returning a channel here
	itemc := make(chan *Item, 1)
	q.Lock()
	q.subs[itemc] = f
	q.Unlock()
	go q.process()

	for {
		select {
		case <-c.Done():
			q.Lock()
			delete(q.subs, itemc)
			q.Unlock()
			return nil
		case item := <-itemc:
			return item
		}
	}
}

// Cancel removes the item from the queue.
func (q *queue) Cancel(c context.Context, item *Item) {

}

// Cancelled returns true if the item is cancelled.
func (q *queue) Cancelled(c context.Context, item *Item) bool {
	return false
}

// helper function that loops through the queue and attempts to
// match the item to a single subscriber.
func (q *queue) process() {
	q.Lock()
	defer q.Unlock()

	// push items to the front of the queue if the ack expires.
	for item, expired := range q.acks {
		if expired.Before(time.Now()) {
			q.list.PushFront(item)
			delete(q.acks, item)
		}
	}

	var next *list.Element
loop:
	for e := q.list.Front(); e != nil; e = next {
		item := e.Value.(*Item)
		for itemc, f := range q.subs {
			if f(item) {
				delete(q.subs, itemc)
				q.list.Remove(e)
				q.acks[item] = time.Now().Add(time.Minute * 5)
				itemc <- item
				break loop
			}
		}
	}
}
