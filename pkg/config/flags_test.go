package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomParseBrowserFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{

		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single flag without value",
			input:    "--no-sandbox",
			expected: []string{"--no-sandbox"},
		},
		{
			name:     "single flag with value",
			input:    "--window-size=1920,1080",
			expected: []string{"--window-size=1920,1080"},
		},
		{
			name:  "multiple flags",
			input: "--host-resolver-rules=MAP fonts.googleapis.com 127.0.0.1, MAP fonts.gstatic.com 127.0.0.1 --no-sandbox --disable-gpu=true",
			expected: []string{
				"--host-resolver-rules=MAP fonts.googleapis.com 127.0.0.1, MAP fonts.gstatic.com 127.0.0.1",
				"--no-sandbox",
				"--disable-gpu=true",
			},
		},
		{
			name:     "flags with special characters",
			input:    `--user-agent="My Browser/1.0 (Test)" --proxy-server="http://proxy:8080"`,
			expected: []string{`--user-agent="My Browser/1.0 (Test)"`, `--proxy-server="http://proxy:8080"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := customParseBrowserFlags(tc.input)
			require.ElementsMatch(t, tc.expected, result)
		})
	}
}
