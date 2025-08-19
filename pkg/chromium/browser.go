package chromium

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Browser struct {
	// Binary is a path to the browser's binary on the file-system.
	Binary string
}

func NewBrowser(
	binary string,
) (*Browser, error) {
	return &Browser{
		Binary: binary,
	}, nil
}

// GetVersion finds the version of the browser.
func (b *Browser) GetVersion(ctx context.Context) (string, error) {
	version, err := exec.CommandContext(ctx, b.Binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version of browser: %w", err)
	}
	return string(bytes.TrimSpace(version)), nil
}
