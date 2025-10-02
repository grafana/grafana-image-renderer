package healthcheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:   "healthcheck",
		Usage:  "Check the server is running and is healthy.",
		Flags:  config.ServerFlags(),
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	serverConfig, err := config.ServerConfigFromCommand(c)
	if err != nil {
		return fmt.Errorf("failed to parse server config: %w", err)
	}

	scheme := "http"
	if serverConfig.CertificateFile != "" {
		scheme = "https"
	}

	var host string
	if strings.HasPrefix(serverConfig.Addr, ":") {
		host = "localhost" + serverConfig.Addr
	} else if !strings.HasPrefix(serverConfig.Addr, "http://") && !strings.HasPrefix(serverConfig.Addr, "https://") {
		host = serverConfig.Addr
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s://%s/healthz", scheme, host), nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	req.Header = http.Header{
		"User-Agent": []string{"grafana-image-renderer/Grafana Labs"},
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform health check request: %w", err)
	}
	// We don't care about the body, so we can ignore closing errors, too.
	_ = resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check request returned non-2xx status code: %d", resp.StatusCode)
	}

	slog.InfoContext(ctx, "OK")
	return nil
}
