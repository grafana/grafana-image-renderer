package server

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run the server part of the service.",
		Action: func(ctx context.Context, c *cli.Command) error {
			return errors.New("TODO")
		},
	}
}
