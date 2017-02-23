package main

import (
	"context"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"time"

	"github.com/cncd/pipeline/pipeline"
	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/backend/docker"
	"github.com/cncd/pipeline/pipeline/interrupt"
	"github.com/cncd/pipeline/pipeline/multipart"
	"github.com/cncd/pipeline/pipeline/rpc"

	_ "github.com/joho/godotenv/autoload"
	"github.com/tevino/abool"
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

	sigterm := abool.New()
	ctx := context.Background()
	ctx = interrupt.WithContextFunc(ctx, func() {
		println("ctrl+c received, terminating process")
		sigterm.Set()
	})

	for {
		if sigterm.IsSet() {
			return nil
		}
		if err := run(ctx, client); err != nil {
			return err
		}
	}
}

func run(ctx context.Context, client rpc.Peer) error {
	log.Println("pipeline: request next execution")

	// get the next job from the queue
	work, err := client.Next(ctx)
	if err != nil {
		return err
	}
	if work == nil {
		return nil
	}
	log.Printf("pipeline: received next execution: %s", work.ID)

	// new docker engine
	engine, err := docker.NewEnv()
	if err != nil {
		return err
	}

	timeout := time.Hour
	if work.Timeout != 0 {
		timeout = time.Duration(work.Timeout) * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		ok, _ := client.Notify(ctx, work.ID)
		if ok {
			log.Printf("pipeline: cancel signal received: %s", work.ID)
			cancel()
		} else {
			log.Printf("pipeline: cancel channel closed: %s", work.ID)
		}
	}()

	state := rpc.State{}
	state.Started = time.Now().Unix()
	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("pipeline: error updating pipeline status: %s: %s", work.ID, err)
	}

	defaultLogger := pipeline.LogFunc(func(proc *backend.Step, rc multipart.Reader) error {
		part, rerr := rc.NextPart()
		if rerr != nil {
			return rerr
		}
		writer := rpc.NewLineWriter(client, work.ID, proc.Name)
		io.Copy(writer, part)

		defer func() {
			log.Printf("pipeline: finish uploading logs: %s: step %s", work.ID, proc.Alias)
		}()

		part, rerr = rc.NextPart()
		if rerr != nil {
			return nil
		}
		mime := part.Header().Get("Content-Type")
		if serr := client.Save(context.Background(), work.ID, mime, part); serr != nil {
			log.Printf("pipeline: cannot upload artifact: %s: %s: %s", work.ID, mime, serr)
		}
		return nil
	})

	err = pipeline.New(work.Config,
		pipeline.WithContext(ctx),
		pipeline.WithLogger(defaultLogger),
		pipeline.WithTracer(pipeline.DefaultTracer),
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
		}
		if state.ExitCode == 0 {
			state.ExitCode = 1
		}
	}

	log.Printf("pipeline: execution complete: %s", work.ID)

	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("Pipeine: error updating pipeline status: %s: %s", work.ID, err)
	}

	return nil
}
