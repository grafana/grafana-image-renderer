package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/grafana/grafana-image-renderer/cmd"
	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	// Until the command actually does some work to parse log level and whatnot, we will log everything in logfmt format for DEBUG level.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})))

	_, err := maxprocs.Set(
		// We use maxprocs over automaxprocs because we need a new minimum value.
		// 2 is the absolute minimum we can handle, because we use multiple goroutines many places for timeouts.
		maxprocs.Min(2),
		maxprocs.Logger(maxProcsLog))
	if err != nil {
		slog.Info("failed to set GOMAXPROCS", "err", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cmd.NewRootCmd().Run(ctx, args); err != nil && !errors.Is(err, context.Canceled) {
		slog.ErrorContext(ctx, "error running command", "err", err)
		return 1
	}

	return 0
}

func maxProcsLog(format string, args ...interface{}) {
	slog.Info(fmt.Sprintf(format, args...), "component", "automaxprocs")
}
