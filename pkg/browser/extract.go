package browser

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
)

func Extract(ctx context.Context) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	browserFS, err := OpenFS()
	if err != nil {
		return "", fmt.Errorf("failed to open embedded fs: %w", err)
	}

	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	err = fs.WalkDir(browserFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if p == "." {
			return nil
		}
		if d.IsDir() {
			if err := os.Mkdir(path.Join(tmp, p), 0o755); err != nil {
				return fmt.Errorf("failed to mkdir %q: %w", p, err)
			}
			return nil
		}

		contents, err := browserFS.ReadFile(p)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", p, err)
		}

		if err := os.WriteFile(path.Join(tmp, p), contents, 0o755); err != nil {
			return fmt.Errorf("failed to write %q: %w", p, err)
		}
		return nil
	})
	if err != nil {
		return tmp, fmt.Errorf("failed to extract to %q: %w", tmp, err)
	}

	return tmp, nil
}
