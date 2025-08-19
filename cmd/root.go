package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/grafana/grafana-image-renderer/cmd/server"
	"github.com/urfave/cli/v3"
)

func NewRootCmd() *cli.Command {
	return &cli.Command{
		Name:    "grafana-image-renderer",
		Usage:   "A service for Grafana to render images and documents from Grafana websites.",
		Version: calculateVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "The minimum level to log at (enum: debug, info, warn, error)",
				DefaultText: "info",
				Validator: func(s string) error {
					if s != "debug" && s != "info" && s != "warn" && s != "error" {
						return fmt.Errorf("invalid log level: %s", s)
					}
					return nil
				},
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			var leveler slog.Leveler
			switch c.String("log-level") {
			case "debug":
				leveler = slog.LevelDebug
			case "info":
				leveler = slog.LevelInfo
			case "warn":
				leveler = slog.LevelWarn
			case "error":
				leveler = slog.LevelError
			default:
				return ctx, fmt.Errorf("invalid log level: %s", c.String("log-level"))
			}
			slog.SetDefault(slog.New(slog.NewTextHandler(c.Writer, &slog.HandlerOptions{AddSource: true, Level: leveler})))

			return ctx, nil
		},
		Commands: []*cli.Command{
			server.NewCmd(),
		},
	}
}

func calculateVersion() string {
	var (
		revision string = "<no vcs info in build>"
		time     string
		modified = true
	)

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return revision + "+dirty"
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			time = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		default:
			continue
		}
	}

	if modified {
		revision += "+dirty"
	}
	if time != "" {
		revision += " (committed " + time + ")"
	}
	if info.GoVersion != "" {
		revision += " (" + info.GoVersion + ")"
	}
	return revision
}
