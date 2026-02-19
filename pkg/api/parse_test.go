package api

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		raw           string
		expected      time.Duration
		expectedError string
	}{
		// Valid cases - numeric strings (interpreted as seconds)
		{
			name:     "valid numeric string 0",
			raw:      "0",
			expected: 0,
		},
		{
			name:     "valid numeric string 10",
			raw:      "10",
			expected: 10 * time.Second,
		},
		{
			name:     "valid numeric string max safe",
			raw:      fmt.Sprintf("%d", maxTimeoutSeconds),
			expected: time.Duration(maxTimeoutSeconds) * time.Second,
		},

		// Valid cases - duration strings
		{
			name:     "valid duration string 10s",
			raw:      "10s",
			expected: 10 * time.Second,
		},
		{
			name:     "valid duration string 1m",
			raw:      "1m",
			expected: 1 * time.Minute,
		},
		{
			name:     "valid duration string 1h",
			raw:      "1h",
			expected: 1 * time.Hour,
		},
		{
			name:     "valid duration string 500ms",
			raw:      "500ms",
			expected: 500 * time.Millisecond,
		},

		// Invalid cases - numeric overflow
		{
			name:          "numeric overflow",
			raw:           fmt.Sprintf("%d", maxTimeoutSeconds+1),
			expectedError: fmt.Sprintf("timeout %d seconds out of representable range", maxTimeoutSeconds+1),
		},
		{
			name:          "int64 min",
			raw:           fmt.Sprintf("%d", int64(math.MinInt64)),
			expectedError: fmt.Sprintf("missing unit in duration \"%d\"", int64(math.MinInt64)),
		},
		{
			name:          "int64 max",
			raw:           fmt.Sprintf("%d", int64(math.MaxInt64)),
			expectedError: fmt.Sprintf("timeout %d seconds out of representable range", int64(math.MaxInt64)),
		},
		{
			name:          "uint64 max",
			raw:           fmt.Sprintf("%d", uint64(math.MaxUint64)),
			expectedError: "value out of range",
		},

		// Invalid cases - negative numeric
		{
			name:          "numeric negative",
			raw:           "-1",
			expectedError: "invalid timeout \"-1\": time: missing unit in duration \"-1\"",
		},
		// Invalid cases - negative duration
		{
			name:          "duration negative",
			raw:           "-1s",
			expectedError: "invalid timeout \"-1s\": negative value",
		},

		// Invalid cases - invalid format
		{
			name:          "invalid format",
			raw:           "invalid",
			expectedError: "invalid timeout \"invalid\": time: invalid duration \"invalid\"",
		},
		{
			name:          "empty string",
			raw:           "",
			expectedError: "invalid timeout \"\": time: invalid duration \"\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseTimeout(tt.raw)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
