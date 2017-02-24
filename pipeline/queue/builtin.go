package queue

import (
	"container/list"
	"context"
	"log"
	"runtime"
	"sync"
	"time"
)

type entry struct {
	item     *Item
	done     chan bool
	error    error
	deadline time.Time
}

type worker struct {
	filter  Filter
	channel chan *Item
}

type queue struct {
	sync.Mutex

	workers map[*worker]struct{}
	running map[string]*entry
	pending *list.List
}

func newQueue() *queue {
	return &queue{
		workers: map[*worker]struct{}{},
		running: map[string]*entry{},
		pending: list.New(),
	}
}

// Push pushes an item to the tail of this queue.
func (q *queue) Push(c context.Context, item *Item) {
	q.Lock()
	q.pending.PushBack(item)
	q.Unlock()
	go q.process()
}

// Poll retrieves and removes the head of this queue.
func (q *queue) Poll(c context.Context, f Filter) *Item {
	q.Lock()
	w := &worker{
		channel: make(chan *Item, 1),
		filter:  f,
	}
	q.workers[w] = struct{}{}
	q.Unlock()
	go q.process()

	for {
		select {
		case <-c.Done():
			q.Lock()
			delete(q.workers, w)
			q.Unlock()
			return nil
		case item := <-w.channel:
			return item
		}
	}
}

// Done signals that the item is done executing.
func (q *queue) Done(c context.Context, id string) {
	q.Error(c, id, nil)
}

// Error signals that the item is done executing with error.
func (q *queue) Error(c context.Context, id string, err error) {
	q.Lock()
	state, ok := q.running[id]
	if ok {
		state.error = err
		close(state.done)
		delete(q.running, id)
	}
	q.Unlock()
}

// Wait waits until the item is done executing.
func (q *queue) Wait(c context.Context, id string) error {
	q.Lock()
	state := q.running[id]
	q.Unlock()
	if state != nil {
		select {
		case <-c.Done():
		case <-state.done:
			return state.error
		}
	}
	return nil
}

// helper function that loops through the queue and attempts to
// match the item to a single subscriber.
func (q *queue) process() {
	defer func() {
		// the risk of panic is low. This code can probably be removed
		// once the code has been used in real world installs without issue.
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("queue: unexpected panic: %v\n%s", err, buf)
		}
	}()

	q.Lock()
	defer q.Unlock()

	// TODO(bradrydzewski) move this to a helper function
	// push items to the front of the queue if the item expires.
	for item, state := range q.running {
		if time.Now().After(state.deadline) {
			q.pending.PushFront(item)
			delete(q.running, item)
			close(state.done)
		}
	}

	var next *list.Element
loop:
	for e := q.pending.Front(); e != nil; e = next {
		item := e.Value.(*Item)
		for w := range q.workers {
			if w.filter(item) {
				delete(q.workers, w)
				q.pending.Remove(e)

				// TODO(bradrydzewski) split timeout calculation to helper func.
				timeout := time.Minute * time.Duration(item.Timeout)
				q.running[item.ID] = &entry{
					item: item,
					done: make(chan bool),

					// TODO(bradrydzewski) right now we only add a 1 minute buffer
					// to the timeout which may not be enough time.
					deadline: time.Now().Add(time.Minute).Add(timeout),
				}

				w.channel <- item
				break loop
			}
		}
	}
}
