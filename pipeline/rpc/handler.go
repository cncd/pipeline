package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	websocketrpc "github.com/sourcegraph/jsonrpc2/websocket"
)

// ErrNoSuchMethod is returned when the name rpc method does not exist.
var ErrNoSuchMethod = errors.New("No such rpc method")

type handler struct {
	client Client
}

// Handle returns a new http.Handler that reads json rpc requests
// and invokes the client methods.
func Handle(client Client) http.Handler {
	return &handler{client}
}

// ServeHTTP implements the http.Handler interface.
func (s *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	conn := jsonrpc2.NewConn(ctx,
		websocketrpc.NewObjectStream(c),
		jsonrpc2.HandlerWithError(s.router),
	)
	defer func() {
		cancel()
		conn.Close()
	}()
	<-conn.DisconnectNotify()
}

// router invokes the named json rpc methods. If the method name is invalid an
// error message is returned.
func (s *handler) router(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	switch req.Method {
	case "next":
		return s.next(ctx, req)
	case "notify":
		return s.notify(ctx, req)
	case "update":
		return s.update(req)
	case "log":
		return s.log(req)
	case "save":
		return s.save(req)
	default:
		return nil, ErrNoSuchMethod
	}
}

func (s *handler) next(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	return s.client.Next(ctx)
}

func (s *handler) notify(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	var id string
	err := json.Unmarshal([]byte(*req.Params), &id)
	if err != nil {
		return nil, err
	}
	return s.client.Notify(ctx, id)
}

func (s *handler) update(req *jsonrpc2.Request) (interface{}, error) {
	in := struct {
		ID    string `json:"id"`
		State State  `json:"state"`
	}{}
	err := json.Unmarshal([]byte(*req.Params), &in)
	if err != nil {
		return nil, err
	}
	return nil, s.client.Update(context.Background(), in.ID, in.State)
}

func (s *handler) log(req *jsonrpc2.Request) (interface{}, error) {
	in := struct {
		ID   string `json:"id"`
		Line string `json:"line"`
	}{}
	err := json.Unmarshal([]byte(*req.Params), &in)
	if err != nil {
		return nil, err
	}
	return nil, s.client.Log(context.Background(), in.ID, in.Line)
}

func (s *handler) save(req *jsonrpc2.Request) (interface{}, error) {
	in := struct {
		ID   string `json:"id"`
		Mime string `json:"mime"`
		Data []byte `json:"data"`
	}{}
	err := json.Unmarshal([]byte(*req.Params), &in)
	if err != nil {
		return nil, err
	}
	return nil, s.client.Save(context.Background(), in.ID, in.Mime, bytes.NewBuffer(in.Data))
}
