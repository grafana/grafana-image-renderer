package chromium

import (
	"context"
	"fmt"
	"os/exec"
)

type BrowserOpts struct {
	// Binary is a path to the browser's binary on the file-system.
	Binary string
}

// GetVersion finds the version of the browser.
func (opts *BrowserOpts) GetVersion(ctx context.Context) (string, error) {
	version, err := exec.CommandContext(ctx, opts.Binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version of browser: %w", err)
	}
	return string(version), nil
}
