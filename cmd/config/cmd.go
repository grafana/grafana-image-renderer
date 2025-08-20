package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Display the configuration of the service.",
		Description: `Read and recreate a merged, single view configuration file from the files used by the application.
This reads the config.yaml, config.yml, and config.json files, if they exist.
The first file takes precedence, then the second, then the third; this behaves like a cascading configuration system.

There is no verification that the values in the configurations actually work and mean anything in the application.`,
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	base := make(map[any]any)
	for _, file := range []string{"config.json", "config.yml", "config.yaml"} {
		tree, err := readFile(file)
		if err != nil {
			return fmt.Errorf("failed to read config file %q: %w", file, err)
		}
		if err := mergeTrees(base, tree); err != nil {
			return fmt.Errorf("failed to merge config file %q: %w", file, err)
		}
	}

	marshalled, err := yaml.Marshal(base)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, _ = fmt.Fprint(c.Writer, string(marshalled))
	return nil
}

func readFile(name string) (map[any]any, error) {
	contents, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[any]any), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to ReadFile: %w", err)
	}

	var tree map[any]any
	if err := yaml.Unmarshal(contents, &tree); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return tree, nil
}

// mergeTrees applies all values in src on top of dst.
// It is a deep merge, i.e. nested maps will be merged recursively.
func mergeTrees(dst, src map[any]any) error {
	for key, value := range src {
		if _, ok := dst[key]; ok {
			// If we have no work to do, just put it in there. There is no need to merge anything.
			dst[key] = value
			continue
		}

		if srcMap, srcIsMap := value.(map[any]any); srcIsMap {
			if dstMap, ok := dst[key].(map[any]any); ok {
				if err := mergeTrees(dstMap, srcMap); err != nil {
					return fmt.Errorf("failed to merge maps for key %v: %w", key, err)
				}
			} else {
				return fmt.Errorf("cannot merge map into non-map value for key %v", key)
			}
		}

		dst[key] = value
	}
	return nil
}
