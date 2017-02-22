package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/cncd/pipeline/pipeline"
	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/backend/docker"
	"github.com/cncd/pipeline/pipeline/multipart"
	"github.com/cncd/pipeline/pipeline/rpc"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "piped"
	app.Usage = "piped stars a pipeline execution daemon"
	app.Action = start
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "endpoint",
			EnvVar: "PIPED_ENDPOINT",
			Value:  "ws://localhost:9999",
		},
		cli.StringFlag{
			Name:   "token",
			EnvVar: "PIPED_TOKEN",
		},
		cli.DurationFlag{
			Name:   "backoff",
			EnvVar: "PIPED_BACKOFF",
			Value:  time.Second * 15,
		},
		cli.IntFlag{
			Name:   "retry-limit",
			EnvVar: "PIPED_RETRY_LIMIT",
			Value:  math.MaxInt32,
		},
		cli.StringFlag{
			Name:   "platform",
			EnvVar: "PIPED_PLATFORM",
			Value:  "linux/amd64",
		},
	}

	// TODO DELETE THIS
	app.Commands = []cli.Command{
		cli.Command{
			Name:   "serve",
			Usage:  "test server",
			Action: serve,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func start(c *cli.Context) error {
	endpoint, err := url.Parse(
		c.String("endpoint"),
	)
	if err != nil {
		return err
	}

	client, err := rpc.NewClient(
		endpoint.String(),
		rpc.WithRetryLimit(
			c.Int("retry-limit"),
		),
		rpc.WithBackoff(
			c.Duration("backoff"),
		),
	)
	if err != nil {
		return err
	}
	defer client.Close()

	for {
		if err := run(client); err != nil {
			return err
		}
	}
}

func run(client rpc.Peer) error {
	log.Println("pipeline: request next execution")

	// get the next job from the queue
	work, err := client.Next(context.Background())
	if err != nil {
		return err
	}
	if work == nil {
		return nil
	}
	log.Println("pipeline: received next execution")

	// new docker engine
	engine, err := docker.NewEnv()
	if err != nil {
		return err
	}

	timeout := time.Hour
	// if work.Timeout != 0 {
	// 	timeout = time.Duration(work.Timeout) * time.Minute
	// }

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// TODO handle interrupt

	// TODO handle cancel request
	// go func() {
	//   client.Notify(c, id)
	// }()

	log.Println("pipeline: executing")

	state := rpc.State{}
	state.Started = time.Now().Unix()
	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("Pipeine: error updating pipeline status: %s", err)
	}

	defaultLogger := pipeline.LogFunc(func(proc *backend.Step, rc multipart.Reader) error {
		part, rerr := rc.NextPart()
		if rerr != nil {
			return rerr
		}
		// buf := bufio.NewReaderSize(part, 1000000)
		io.Copy(&lineReader{client: client, id: work.ID}, part)
		return nil
	})

	err = pipeline.New(work.Config,
		pipeline.WithContext(ctx),
		pipeline.WithLogger(defaultLogger),
		pipeline.WithTracer(defaultTracer),
		pipeline.WithEngine(engine),
	).Run()

	state.Finished = time.Now().Unix()
	state.Exited = true
	if err != nil {
		state.Error = err.Error()
		if xerr, ok := err.(*pipeline.ExitError); ok {
			state.ExitCode = xerr.Code
		}
		if xerr, ok := err.(*pipeline.OomError); ok {
			state.ExitCode = xerr.Code
			if state.ExitCode == 0 {
				state.ExitCode = 1
			}
		}
	}

	log.Println("pipeline: execution complete")

	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("Pipeine: error updating pipeline status: %s", err)
	}

	return nil
}

type lineReader struct {
	client rpc.Peer
	line   int
	id     string
}

func (r *lineReader) Write(p []byte) (n int, err error) {
	parts := bytes.Split(p, []byte{'\n'})
	for _, part := range parts {
		r.line++
		r.client.Log(context.Background(), r.id, string(part))
	}
	return len(p), nil
}

var defaultTracer = pipeline.TraceFunc(func(state *pipeline.State) error {
	if !state.Process.Exited {
		state.Pipeline.Step.Environment["CI_BUILD_STATUS"] = "success"
		state.Pipeline.Step.Environment["CI_BUILD_STARTED"] = strconv.FormatInt(state.Pipeline.Time, 10)
		state.Pipeline.Step.Environment["CI_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
		if state.Pipeline.Error != nil {
			state.Pipeline.Step.Environment["CI_BUILD_STATUS"] = "failure"
		}
	}
	return nil
})
