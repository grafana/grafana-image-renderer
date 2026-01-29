package config

import (
	"strings"
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

func FuzzCustomParseBrowserFlags(f *testing.F) {
	f.Skip("Skipping fuzz test, enable manually when needed")

	f.Add("")
	f.Add("--no-sandbox")
	f.Add("--window-size=1920,1080")
	f.Add("--host-resolver-rules=MAP fonts.googleapis.com 127.0.0.1, MAP fonts.gstatic.com 127.0.0.1 --no-sandbox --disable-gpu=true")
	f.Add(`--user-agent="My Browser/1.0 (Test)" --proxy-server="http://proxy:8080"`)
	f.Add("-")
	f.Add("--")
	f.Add("---")
	f.Add("----")
	f.Add("-----")
	f.Add("-- --")
	f.Add("--flag --")
	f.Add("-- --flag")
	f.Add("   --flag   ")
	f.Add("\t--flag\t")
	f.Add("--flag\n--other")
	f.Add("--flag\r\n--other")
	f.Add("no-dashes-here")
	f.Add("single-dash-only")
	f.Add("-single-dash")
	f.Add("--=value")
	f.Add("--flag=")
	f.Add("--flag==double")
	f.Add("--flag=value=with=equals")
	f.Add("--flag=value --flag=value")
	f.Add("--a --b --c --d --e --f")
	f.Add("--flag=value-with-dashes")
	f.Add("--flag=value--with--double--dashes")
	f.Add("prefix --flag suffix")
	f.Add("--flag=中文")
	f.Add("--flag=émoji🎉")
	f.Add("--flag='single quotes'")
	f.Add(`--flag="double quotes"`)
	f.Add("--flag=`backticks`")
	f.Add("--flag=$(command)")
	f.Add("--flag=${VAR}")
	f.Add("--flag=a\x00b")
	f.Add("--flag=a\tb")
	f.Add("--very-long-flag-name=value")
	f.Add("--f=v")
	f.Add("--1=2")
	f.Add("--_underscore=value")
	f.Add("--UPPERCASE=value")
	f.Add("--MixedCase=Value")
	f.Add("--flag=http://example.com:8080")
	f.Add("--flag=/path/to/file")
	f.Add("--flag=C:\\Windows\\Path")
	f.Add("--flag=a,b,c,d")
	f.Add("--flag=a;b;c;d")
	f.Add("--flag=a|b|c|d")
	f.Add("--a=1 --b=2 --c=3")
	f.Add("  --flag  ")
	f.Add("--flag  --other")
	f.Add("text before --flag text after")
	f.Add("--flag=value\twith\ttabs")
	f.Add("--flag=line1\nline2")
	f.Add("--flag=\x00\x01\x02")
	f.Add("--flag=日本語テスト")
	f.Add("--flag=العربية")
	f.Add("--flag=🎉🎊🎈")
	f.Add(strings.Repeat("--flag ", 100))
	f.Add("--" + strings.Repeat("a", 1000))

	f.Fuzz(func(t *testing.T, input string) {
		result := customParseBrowserFlags(input)

		// Invariant: If input contains no "--", result must be empty
		if !strings.Contains(input, "--") && len(result) > 0 {
			t.Errorf("input %q has no '--' but got %d flags: %v", input, len(result), result)
		}

		// Invariant: Each result element must be non-empty
		for i, flag := range result {
			if strings.TrimSpace(flag) == "" {
				t.Errorf("flag at index %d is empty or whitespace-only: %q", i, flag)
			}
		}

		// Invariant: Each result element must be derived from the input
		for i, flag := range result {
			// The flag content should appear somewhere in the input
			if !strings.Contains(input, flag) {
				// It might be trimmed, so also check without trim
				trimmedInput := strings.TrimSpace(input)
				if !strings.Contains(trimmedInput, flag) && !strings.Contains(input, flag) {
					t.Errorf("flag at index %d (%q) content not found in input %q", i, flag, input)
				}
			}
		}

		// Invariant: Result length is bounded by the number of positions where "--" can start
		// Each "--" pattern starting position can produce at most one flag
		// For overlapping dashes like "----", there can be multiple "--" start positions
		maxPossibleFlags := 0
		for i := 0; i < len(input)-1; i++ {
			if input[i] == '-' && input[i+1] == '-' {
				maxPossibleFlags++
			}
		}
		if len(result) > maxPossibleFlags {
			t.Errorf("too many flags (%d) for input with %d '--' start positions", len(result), maxPossibleFlags)
		}
	})
}
