//go:build !linux

package sandbox

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:            "_internal_sandbox",
		SkipFlagParsing: true,
		Action: func(ctx context.Context, c *cli.Command) error {
			return fmt.Errorf("unsupported on your system")
		},
	}
}
