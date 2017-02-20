package main

import (
	"context"
	"os"
	"os/signal"
)

// WithInterrupt returns a context that is cancelled when an
// interrupt signal is received by the application.
func WithInterrupt(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case <-c:
			println("ctrl+c received, shutting down")
			cancel()
		}
	}()

	return ctx
}
