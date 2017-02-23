package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/rpc"
	"github.com/urfave/cli"
)

// this section implements a dummy server that can send
// builds to the piped polling agent.

func serve(c *cli.Context) error {
	handler := new(handler)
	go handler.start()
	server := rpc.NewServer(handler)

	log.Println("starting server on port :9999")
	return http.ListenAndServe(":9999", server)
}

type handler struct {
	queue chan *rpc.Pipeline
}

func (h *handler) start() {
	var i = 0
	h.queue = make(chan *rpc.Pipeline)
	for {
		i++
		next := &rpc.Pipeline{
			Config: &backend.Config{
				Stages: []*backend.Stage{
					{
						Name:  fmt.Sprintf("test_stage_%d", i),
						Alias: "test_stage",
						Steps: []*backend.Step{
							{
								Name:        fmt.Sprintf("test_step_%d", i),
								Alias:       "test_step",
								Image:       "golang:1.7",
								Entrypoint:  []string{"/bin/sh", "-c"},
								Environment: map[string]string{"CI": "true"},
								Command: []string{
									strings.Join([]string{
										"set -x",
										"echo hello",
										"sleep 5",
										"echo world",
										"sleep 5",
										"echo hola",
										"echo mundo",
										"echo done!",
									}, "\n"),
								},
								OnSuccess: true,
							},
						},
					},
				},
			},
		}
		next.Timeout = 60
		next.ID = strconv.Itoa(i)

		h.queue <- next
		<-time.After(45 * time.Second)
	}
}

func (h *handler) Next(c context.Context) (*rpc.Pipeline, error) {
	select {
	case next := <-h.queue:
		return next, nil
	case <-c.Done():
		return nil, nil
	}
}

func (*handler) Notify(c context.Context, id string) (bool, error) {

	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt)
	defer signal.Stop(sigterm)

	select {
	case <-c.Done():
		println("pipeline: cancel: interrupt")
		return false, nil
	case <-sigterm:
		println("pipeline: cancel: received")
		return true, nil
	case <-time.After(120 * time.Second):
		println("pipeline: cancel: timeout")
		return false, nil
	}
}

func (*handler) Update(c context.Context, id string, state rpc.State) error {
	log.Printf("pipeline: update %s: exited=%v, exit_code=%d", id, state.Exited, state.ExitCode)
	return nil
}
func (*handler) Save(c context.Context, id, mime string, file io.Reader) error { return nil }
func (*handler) Log(c context.Context, id string, line *rpc.Line) error {
	log.Printf("pipeline: logs %s: %s", id, line)
	return nil
}
