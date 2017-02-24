package queue

import (
	"context"
	"fmt"
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
	item := &Item{ID: "1"}

	q := newQueue()
	q.Push(context.Background(), item)
	if q.pending.Len() != 1 {
		t.Errorf("Expect item in pending list")
	}
	if q.pending.Front().Value != item {
		t.Errorf("Expect item in pending list")
	}

	got := q.Poll(context.Background(), filterTrue)
	if got == nil {
		t.Errorf("Expect item from head of queue")
	} else if got.ID != "1" {
		t.Errorf("Incorrect item id %s", item.ID)
	}

	if q.pending.Len() != 0 {
		t.Errorf("Expect empty pending list")
	}
	if len(q.workers) != 0 {
		t.Errorf("Expect empty subscriber map")
	}
	if len(q.running) != 1 {
		t.Errorf("Expect item in running list")
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		q.Wait(context.Background(), got.ID)
		wg.Done()
	}()
	go func() {
		q.Wait(context.Background(), got.ID)
		wg.Done()
	}()

	q.Done(context.Background(), got.ID)
	if len(q.running) != 0 {
		t.Errorf("Expect item removed from running list")
	}
	wg.Wait()
}

func TestQueueError(t *testing.T) {
	item := &Item{ID: "1"}

	q := newQueue()
	q.Push(context.Background(), item)

	got := q.Poll(context.Background(), filterTrue)

	var err = fmt.Errorf("some random error")
	var err1 error
	var err2 error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err1 = q.Wait(context.Background(), got.ID)
		wg.Done()
	}()
	go func() {
		err2 = q.Wait(context.Background(), got.ID)
		wg.Done()
	}()

	q.Error(context.Background(), got.ID, err)
	wg.Wait()

	if err1 != err {
		t.Errorf("Expect error returned from Wait")
	}
	if err2 != err {
		t.Errorf("Expect error returned from Wait")
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

	if q.pending.Len() != 1 {
		t.Errorf("Expect item in pending list")
	}
	if len(q.workers) != 0 {
		t.Errorf("Expect list of subscribers is zero")
	}
}
