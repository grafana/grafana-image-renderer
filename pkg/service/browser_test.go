package service

import (
	"testing"
)

func TestExtractSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		url          string
		expectedSlug string
		expectError  bool
	}{
		{
			name:         "simple grafana cloud url",
			url:          "https://myorg.grafana.net",
			expectedSlug: "myorg",
			expectError:  false,
		},
		{
			name:         "cloud url with path and query params",
			url:          "https://myorg.grafana.net/d/abc123/my-dashboard?orgId=1&from=now-6h&to=now",
			expectedSlug: "myorg",
			expectError:  false,
		},
		{
			name:         "cloud url with numeric slug",
			url:          "https://org123.grafana.net/render/d/abc",
			expectedSlug: "org123",
			expectError:  false,
		},
		{
			name:         "non-cloud valid url - different domain",
			url:          "https://grafana.example.com/d/abc123",
			expectedSlug: "",
			expectError:  false,
		},
		{
			name:         "non-cloud valid url - localhost",
			url:          "http://localhost:3000/d/abc123",
			expectedSlug: "",
			expectError:  false,
		},
		{
			name:         "non-cloud valid url - grafana.com instead of .net",
			url:          "https://myorg.grafana.com/d/abc123",
			expectedSlug: "",
			expectError:  false,
		},
		{
			name:         "invalid url - malformed",
			url:          "://not-a-valid-url",
			expectedSlug: "",
			expectError:  true,
		},
		{
			name:         "invalid url - contains control characters",
			url:          "https://example\x00.com",
			expectedSlug: "",
			expectError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			slug, err := extractSlug(tc.url)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if slug != tc.expectedSlug {
				t.Errorf("expected slug %q, got %q", tc.expectedSlug, slug)
			}
		})
	}
}
