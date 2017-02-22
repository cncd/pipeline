package rpc

import (
	"context"
	"io"

	"github.com/cncd/pipeline/pipeline/backend"
)

type (
	// Filter defines filters for fetching items from the queue.
	Filter struct {
		Platform string `json:"platform"`
	}

	// State defines the pipeline state.
	State struct {
		Exited   bool   `json:"exited"`
		ExitCode int    `json:"exit_code"`
		Started  int64  `json:"started"`
		Finished int64  `json:"finished"`
		Error    string `json:"error"`
	}

	// Pipeline defines the pipeline execution details.
	Pipeline struct {
		ID      string          `json:"id"`
		Timeout int64           `json:"timeout"`
		Config  *backend.Config `json:"config"`
	}
)

// Peer defines a peer-to-peer connection.
type Peer interface {
	// Next returns the next pipeline in the queue.
	Next(c context.Context) (*Pipeline, error)

	// Notify returns true if the pipeline should be cancelled.
	Notify(c context.Context, id string) (bool, error)

	// Update updates the pipeline state.
	Update(c context.Context, id string, state State) error

	// Log writes the pipeline log entry.
	Log(c context.Context, id string, line string) error

	// Save saves the pipeline artifact.
	Save(c context.Context, id, mime string, file io.Reader) error
}
