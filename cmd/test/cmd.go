package test

import (
	"context"
	"io/fs"
	"log/slog"

	"github.com/grafana/grafana-image-renderer/pkg/browser"
	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "test",
		Usage: "testing :)",
		Action: func(ctx context.Context, c *cli.Command) error {
			browserFS, err := browser.OpenFS()
			if err != nil {
				return err
			}

			return fs.WalkDir(browserFS, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				slog.Info("entry", "path", path, "isDir", d.IsDir())
				return nil
			})
		},
	}
}
