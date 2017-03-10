package kubernetes

import (
	"io"

	"github.com/cncd/pipeline/pipeline/backend"
)

type engine struct {
	namespace string
	endpoint  string
	token     string
}

// New returns a new Kubernetes Engine.
func New(namespace, endpoint, token string) backend.Engine {
	return &engine{
		namespace: namespace,
		endpoint:  endpoint,
		token:     token,
	}
}

// Setup the pipeline environment.
func (e *engine) Setup(*backend.Config) error {
	return nil
}

// Start the pipeline step.
func (e *engine) Exec(*backend.Step) error {
	return nil
}

// Kill the pipeline step.
func (e *engine) Kill(*backend.Step) error {
	return nil
}

// Wait for the pipeline step to complete and returns
// the completion results.
func (e *engine) Wait(*backend.Step) (*backend.State, error) {
	return nil, nil
}

// Tail the pipeline step logs.
func (e *engine) Tail(*backend.Step) (io.ReadCloser, error) {
	return nil, nil
}

// Destroy the pipeline environment.
func (e *engine) Destroy(*backend.Config) error {
	return nil
}
