package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/grafana/grafana-image-renderer/cmd"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	// Until the command actually does some work to parse log level and whatnot, we will log everything in logfmt format for DEBUG level.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cmd.NewRootCmd().Run(ctx, args); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("error running command", "err", err)
		return 1
	}

	return 0
}
