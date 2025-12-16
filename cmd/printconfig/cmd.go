package printconfig

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/grafana/grafana-image-renderer/cmd/server"
	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "print-config",
		Usage: "Parse and print the current configuration. Replace 'server' with 'print-config' in the command line.",
		Flags: server.NewCmd().Flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := server.ParseConfig(c)
			if err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}
			spew.Dump(cfg)
			return nil
		},
	}
}
