package config

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

// reconstructFlags extracts all flag values from a command and returns them
// as CLI argument strings (e.g., ["--flag=value", "--other=123"]).
// This includes all values regardless of source (CLI, config file, env var, or default).
// This is useful for cloning a command's configuration to build overrides.
func reconstructFlags(cmd *cli.Command) ([]string, error) {
	flags := []string{}
	for _, flag := range cmd.Flags {
		names := flag.Names()
		if len(names) == 0 {
			return nil, fmt.Errorf("flag %v has no names", flag)
		}
		name := names[0]
		if !cmd.IsSet(name) {
			continue // skip flags that are not explicitly set
		}
		value := flag.Get()
		vals, err := reconstructFlagValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct flag value for %s: %w", name, err)
		}
		for _, v := range vals {
			flags = append(flags, fmt.Sprintf("--%s=%s", name, v))
		}
	}
	return flags, nil
}

// reconstructFlagValue converts a flag value to string representation(s).
// For slice values, it returns multiple strings (one per element).
func reconstructFlagValue(v any) ([]string, error) {
	switch v := v.(type) {
	case []string:
		result := make([]string, len(v))
		copy(result, v)
		return result, nil
	case []any:
		result := []string{}
		for _, item := range v {
			vals, err := reconstructFlagValue(item)
			if err != nil {
				return nil, err
			}
			result = append(result, vals...)
		}
		return result, nil
	default:
		return []string{fmt.Sprintf("%v", v)}, nil
	}
}

// parseOverrideFlags splits a flag string into individual flags. This format is required by the cli library. We cannot use Strings.Fields() because it improperly handles special characters.
// Handles formats like "--flag=value --flag2=value2", "-f=value", and flags with spaces in values.
func parseOverrideFlags(flagsStr string) []string {
	flagsStr = strings.TrimSpace(flagsStr)
	if flagsStr == "" {
		return nil
	}

	var flags []string
	var current strings.Builder
	seenFirstFlag := false

	for i := 0; i < len(flagsStr); i++ {
		ch := flagsStr[i]

		// Check if we're starting a new flag (- or -- at start or after whitespace).
		// Dashes in the middle of values (e.g., "--flag=some-value") are not flag boundaries.
		isAtFlagStart := false
		if ch == '-' {
			atBoundary := i == 0 || flagsStr[i-1] == ' ' || flagsStr[i-1] == '\t'
			if atBoundary {
				isAtFlagStart = true
			}
		}

		if isAtFlagStart && seenFirstFlag && current.Len() > 0 {
			flags = append(flags, strings.TrimSpace(current.String()))
			current.Reset()
		}

		if isAtFlagStart {
			seenFirstFlag = true
		}

		if seenFirstFlag {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		flags = append(flags, strings.TrimSpace(current.String()))
	}

	return flags
}
