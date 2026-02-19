package api

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// maxTimeoutSeconds is the largest number of seconds that can be safely
// multiplied by time.Second without overflowing time.Duration.
const maxTimeoutSeconds = math.MaxInt64 / int64(time.Second)

// parseTimeout parses a timeout string to a time.Duration (or an error).
//
// It returns an error for values that would overflow time.Duration when
// converted to nanoseconds, for negative values, and for invalid formats.
//
// This accepts both plain numbers (interpreted as seconds) and time.Duration strings.
func parseTimeout(raw string) (time.Duration, error) {
	if regexpOnlyNumbers.MatchString(raw) {
		seconds, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("invalid timeout %q: %w", raw, err)
		}
		if seconds < 0 || int64(seconds) > maxTimeoutSeconds {
			return 0, fmt.Errorf("timeout %d seconds out of representable range", seconds)
		}
		return time.Duration(seconds) * time.Second, nil
	}

	dur, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", raw, err)
	}
	if dur < 0 {
		return 0, fmt.Errorf("invalid timeout %q: negative value", raw)
	}
	return dur, nil
}
