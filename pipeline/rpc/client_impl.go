package rpc

import (
	"context"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/cncd/pipeline/pipeline/backend"

	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	websocketrpc "github.com/sourcegraph/jsonrpc2/websocket"
)

type client struct {
	sync.Mutex

	conn     *jsonrpc2.Conn
	done     bool
	retry    int
	backoff  time.Duration
	endpoint string
}

// New returns a new Client.
func New(endpoint string) (Client, error) {
	cli := new(client)
	cli.endpoint = endpoint
	cli.retry = 100
	cli.backoff = 5 * time.Second
	err := cli.open()
	return cli, err
}

// Next returns the next pipeline in the queue.
func (t *client) Next(c context.Context) (*backend.Config, error) {
	res := new(backend.Config)
	err := t.call("next", nil, res)
	return res, err
}

// Notify returns true if the pipeline should be cancelled.
func (t *client) Notify(c context.Context, id string) (bool, error) {
	out := false
	err := t.call("notify", id, &out)
	return out, err
}

// Update updates the pipeline state.
func (t *client) Update(c context.Context, id string, state State) error {
	params := struct {
		ID    string `json:"id"`
		State State  `json:"state"`
	}{id, state}
	return t.call("update", &params, nil)
}

// Log writes the pipeline log entry.
func (t *client) Log(c context.Context, id string, line string) error {
	params := struct {
		ID   string `json:"id"`
		Line string `json:"line"`
	}{id, line}
	return t.call("log", &params, nil)
}

// Save saves the pipeline artifact.
func (t *client) Save(c context.Context, id, mime string, file io.Reader) error {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	params := struct {
		ID   string `json:"id"`
		Mime string `json:"mime"`
		Data []byte `json:"data"`
	}{id, mime, data}
	return t.call("save", params, nil)
}

// Close closes the client connection.
func (t *client) Close() error {
	t.Lock()
	t.done = true
	t.Unlock()
	return t.conn.Close()
}

func (t *client) call(name string, req, res interface{}) (err error) {
	err = t.conn.Call(context.Background(), name, req, res)
	if err == nil {
		return
	}
	for i := 0; i < t.retry; i++ {
		err = t.open()
		if err == nil {
			break
		}
		if err == io.EOF {
			return
		}
		<-time.After(t.backoff)
	}
	if err != nil {
		return
	}
	return t.conn.Call(context.Background(), name, req, res)
}

func (t *client) open() error {
	t.Lock()
	defer t.Unlock()
	if t.done {
		return io.EOF
	}
	conn, _, err := websocket.DefaultDialer.Dial(t.endpoint, nil)
	if err != nil {
		return err
	}
	stream := websocketrpc.NewObjectStream(conn)
	t.conn = jsonrpc2.NewConn(context.Background(), stream, nil)
	return nil
}
