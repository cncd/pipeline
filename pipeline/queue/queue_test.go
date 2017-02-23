package queue

import (
	"context"
	"sync"
	"testing"
)

var (
	filterTrue = func(item *Item) bool {
		return true
	}

	filterFalse = func(item *Item) bool {
		return false
	}

	filterLinux = func(item *Item) bool {
		return true
	}
)

func TestQueuePoll(t *testing.T) {
	q := newQueue()
	q.Push(context.Background(), &Item{ID: "1"})
	item := q.Poll(context.Background(), filterTrue)

	if item == nil {
		t.Errorf("Expect item from head of queue")
	} else if item.ID != "1" {
		t.Errorf("Incorrect item id %s", item.ID)
	}

	if q.list.Len() != 0 {
		t.Errorf("Expect empty item list")
	}
	if len(q.subs) != 0 {
		t.Errorf("Expect empty subscriber map")
	}
	if len(q.acks) != 1 {
		t.Errorf("Expect item in ack list")
	}
	if _, ok := q.acks[item]; !ok {
		t.Errorf("Expect item in ack list matches id")
	}
}

func TestQueuePollExpires(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	q := newQueue()
	q.Push(context.Background(), &Item{ID: "1"})

	var wg sync.WaitGroup
	go func() {
		q.Poll(ctx, filterFalse)
		wg.Done()
	}()

	wg.Add(1)
	cancel()
	wg.Wait()

	if q.list.Len() != 1 {
		t.Errorf("Expect item in list")
	}
	if len(q.subs) != 0 {
		t.Errorf("Expect list of subscribers is zero")
	}
}
