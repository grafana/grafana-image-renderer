package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/grafana/grafana-image-renderer/cmd/config"
	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "healthcheck",
		Usage: "Check the server is running and is healthy.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Usage:   "The address to listen on for HTTP requests.",
				Value:   ":8081",
				Sources: config.FromConfig("server.addr"),
			},
		},
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	addr := c.String("addr")
	if strings.HasPrefix(addr, ":") {
		addr = "http://localhost" + addr
	} else if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	req, err := http.NewRequestWithContext(ctx, "GET", addr+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	req.Header = http.Header{
		"User-Agent": []string{"grafana-image-renderer/Grafana Labs"},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform health check request: %w", err)
	}
	defer func() {
		// We don't care about the body, so we can ignore closing errors, too.
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check request returned non-2xx status code: %d", resp.StatusCode)
	}

	return nil
}
