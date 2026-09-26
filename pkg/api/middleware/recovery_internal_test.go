package middleware

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactedURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "renderKey is masked",
			in:   "/render?renderKey=supersecret&url=http%3A%2F%2Fgrafana%2Fd%2Fabc&width=1000",
			want: "/render?renderKey=-redacted-&url=http%3A%2F%2Fgrafana%2Fd%2Fabc&width=1000",
		},
		{
			name: "no renderKey leaves URL untouched",
			in:   "/render?url=http%3A%2F%2Fgrafana%2Fd%2Fabc&width=1000",
			want: "/render?url=http%3A%2F%2Fgrafana%2Fd%2Fabc&width=1000",
		},
		{
			name: "no query at all",
			in:   "/render",
			want: "/render",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u, err := url.Parse(tc.in)
			require.NoError(t, err, "parsing test URL")
			require.Equal(t, tc.want, redactedURL(u), "redacted URL")
		})
	}
}
