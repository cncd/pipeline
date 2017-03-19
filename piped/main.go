package main

import (
	"context"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"sync"
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

const (
	maxFileUpload = 5000000
	maxLogsUpload = 5000000
)

func main() {
	app := cli.NewApp()
	app.Name = "piped"
	app.Usage = "piped stars a pipeline execution daemon"
	app.Action = start
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "endpoint",
			EnvVar: "PIPED_ENDPOINT,PIPED_SERVER",
			Value:  "ws://localhost:9999",
		},
		cli.StringFlag{
			Name:   "token",
			EnvVar: "PIPED_TOKEN,PIPED_SECRET",
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
		cli.Int64Flag{
			Name:   "upload-limit",
			EnvVar: "PIPED_UPLOAD_LIMIT",
			Value:  math.MaxInt32,
		},
	}
	app.Commands = []cli.Command{
		onceCommand,
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
	filter := rpc.Filter{
		Labels: map[string]string{
			"platform": c.String("platform"),
		},
	}

	client, err := rpc.NewClient(
		endpoint.String(),
		rpc.WithRetryLimit(
			c.Int("retry-limit"),
		),
		rpc.WithBackoff(
			c.Duration("backoff"),
		),
		rpc.WithToken(
			c.String("token"),
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
		if err := run(ctx, client, filter); err != nil {
			return err
		}
	}
}

func run(ctx context.Context, client rpc.Peer, filter rpc.Filter) error {
	log.Println("pipeline: request next execution")

	// get the next job from the queue
	work, err := client.Next(ctx, filter)
	if err != nil {
		return err
	}
	if work == nil {
		return nil
	}
	log.Printf("pipeline: received next execution: %s", work.ID)
	if os.Getenv("SUICIDE_MODE") != "" {
		os.Exit(1)
	}
	// new docker engine
	engine, err := docker.NewEnv()
	if err != nil {
		return err
	}

	timeout := time.Hour
	if minutes := work.Timeout; minutes != 0 {
		timeout = time.Duration(minutes) * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cancelled := abool.New()
	go func() {
		werr := client.Wait(ctx, work.ID)
		if werr != nil {
			cancelled.SetTo(true) // TODO verify error is really an error
			log.Printf("pipeline: cancel signal received: %s: %s", work.ID, werr)
			cancel()
		} else {
			log.Printf("pipeline: cancel channel closed: %s", work.ID)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("pipeline: cancel ping loop: %s", work.ID)
				return
			case <-time.After(time.Minute):
				log.Printf("pipeline: ping queue: %s", work.ID)
				client.Extend(ctx, work.ID)
			}
		}
	}()

	state := rpc.State{}
	state.Started = time.Now().Unix()
	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("pipeline: error updating pipeline status: %s: %s", work.ID, err)
	}

	var uploads sync.WaitGroup
	defaultLogger := pipeline.LogFunc(func(proc *backend.Step, rc multipart.Reader) error {
		part, rerr := rc.NextPart()
		if rerr != nil {
			return rerr
		}
		uploads.Add(1)

		var secrets []string
		for _, secret := range work.Config.Secrets {
			if secret.Mask {
				secrets = append(secrets, secret.Value)
			}
		}
		limitedPart := io.LimitReader(part, maxLogsUpload)
		writer := rpc.NewLineWriter(client, work.ID, proc.Alias, secrets...)
		io.Copy(writer, limitedPart)

		defer func() {
			log.Printf("pipeline: finish uploading logs: %s: step %s", work.ID, proc.Alias)
			uploads.Done()
		}()

		part, rerr = rc.NextPart()
		if rerr != nil {
			return nil
		}
		// TODO should be configurable
		limitedPart = io.LimitReader(part, maxFileUpload)
		mime := part.Header().Get("Content-Type")
		if serr := client.Upload(context.Background(), work.ID, mime, limitedPart); serr != nil {
			log.Printf("pipeline: cannot upload artifact: %s: %s: %s", work.ID, mime, serr)
		} else {
			log.Printf("pipeline: finish uploading artifact: %s: step %s: %s", mime, work.ID, proc.Alias)
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
		if cancelled.IsSet() {
			state.ExitCode = 130
		} else if state.ExitCode == 0 {
			state.ExitCode = 1
		}
	}

	log.Printf("pipeline: execution complete: %s", work.ID)

	uploads.Wait()
	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("Pipeine: error updating pipeline status: %s: %s", work.ID, err)
	}

	err = client.Done(context.Background(), work.ID)
	if err != nil {
		log.Printf("Pipeine: error signaling pipeline done: %s: %s", work.ID, err)
	}

	return nil
}
