package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/cncd/pipeline/pipeline/backend"
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
			Name:   "username",
			EnvVar: "PIPED_USERNAME",
		},
		cli.StringFlag{
			Name:   "password",
			EnvVar: "PIPED_PASSWORD",
		},
		cli.DurationFlag{
			Name:   "timeout",
			EnvVar: "PIPED_TIMEOUT",
			Value:  time.Hour,
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
	// endpoint.User = url.UserPassword(
	// 	c.String("username"),
	// 	c.String("password"),
	// )

	client, err := rpc.New(endpoint.String())
	if err != nil {
		return err
	}
	if closer, ok := client.(io.Closer); ok {
		defer closer.Close()
	}

	go func() {
		client.Next(context.Background())
	}()

	for i := 0; i <= 100; i++ {
		id := strconv.Itoa(i)
		log.Printf("line number %s\n", id)

		if err := client.Log(context.Background(), id, "test"); err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}

//
// this code will get removed. here for testing only
//

func serve(c *cli.Context) error {
	handler := new(handler)
	server := rpc.NewServer(handler)

	log.Println("starting server on port :9999")
	return http.ListenAndServe(":9999", server)
}

type handler struct {
}

func (*handler) Next(c context.Context) (*backend.Config, error) {
	println("BLOCKING FOR NEXT")
	select {
	case <-time.After(20 * time.Second):
		println("GOT NEXT")
	case <-c.Done():
		println("CANCEL")
	}
	return nil, nil
}

func (*handler) Notify(c context.Context, id string) (bool, error)             { return false, nil }
func (*handler) Update(c context.Context, id string, state rpc.State) error    { return nil }
func (*handler) Save(c context.Context, id, mime string, file io.Reader) error { return nil }
func (*handler) Log(c context.Context, id string, line string) error {
	log.Printf("got line %s\n", id)
	return nil
}
