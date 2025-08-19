package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

// ListenAndServe is like http.ListenAndServe, except it also deals with context cancellations.
//
// The function is blocking and will return after the context is cancelled AND a shutdown attempt has been done.
//
// TODO: Do we need a way to specify the shutdown timeout?
func ListenAndServe(parentCtx context.Context, addr string, handler http.Handler) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// TODO: Should we support Unix sockets for testing?
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %q: %w", addr, err)
	}

	server := &http.Server{
		Handler:           handler,
		BaseContext:       func(l net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: time.Second * 3, // try to mitigate Slowloris
	}

	grp, grpCtx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		slog.InfoContext(grpCtx, "serving HTTP traffic", "addr", addr)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server failed: %w", err)
		}
		return nil
	})
	grp.Go(func() error {
		<-parentCtx.Done()
		timeoutCtx, cancelTimeout := context.WithTimeout(context.WithoutCancel(grpCtx), time.Second*5)
		defer cancelTimeout()
		return server.Shutdown(timeoutCtx)
	})

	return grp.Wait()
}
