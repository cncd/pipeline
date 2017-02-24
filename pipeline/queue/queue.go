package queue

import (
	"context"

	"github.com/cncd/pipeline/pipeline/backend"
)

// Queue defines a task queue for scheduling tasks among
// a pool of workers.
type Queue interface {
	// Push pushes an item to the tail of this queue.
	Push(c context.Context, item *Item)

	// Poll retrieves and removes the head of this queue.
	Poll(c context.Context, f Filter)

	// Done signals that the item is done executing.
	Done(c context.Context, id string)

	// Error signals that the item is done executing with error.
	Error(c context.Context, id string)

	// Wait waits until the item is done executing.
	Wait(c context.Context, id string)

	// Pending returns a list of pending items in the queue.
	// Pending() []*Item

	// Running returns a list of running items in the queue.
	// Running() []*Item
}

// Item is an item in the queue.
type Item struct {
	ID      string
	Timeout int64
	Config  *backend.Config
}

// Filter filters items in the queue. If the Filter returns false,
// the Item is skipped and not returned to the Subscriber.
type Filter func(*Item) bool
