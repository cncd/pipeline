package rpc

import (
	"context"
	"io"

	"github.com/cncd/pipeline/pipeline/backend"
)

type (
	// Filter defines filters for fetching items from the queue.
	Filter struct {
		Platform string
	}

	// State defines the pipeline state.
	State struct {
		Exited   bool
		ExitCode int
		Started  int64
		Finished int64
		Error    error
	}
)

// Client defines a pipeline client.
type Client interface {
	// Next returns the next pipeline in the queue.
	Next(c context.Context) (*backend.Config, error)

	// Notify returns true if the pipeline should be cancelled.
	Notify(c context.Context, id string) (bool, error)

	// Update updates the pipeline state.
	Update(c context.Context, id string, state State) error

	// Log writes the pipeline log entry.
	Log(c context.Context, id string, line string) error

	// Save saves the pipeline artifact.
	Save(c context.Context, id, mime string, file io.Reader) error
}
