package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/676
func TestRegression676(t *testing.T) {
	LongTest(t)
	t.Parallel()

	const importCA = `
db="$(mktemp -d)"
certutil -N -d "sql:$db" --empty-password
certutil -A -d "sql:$db" -n test-root -t "C,," -a -i "${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"
certutil -L -d "sql:$db" -n test-root
`

	exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-ec"}, []string{importCA})
	require.Zero(t, exitCode, "could not import a CA from the image bundle into NSS:\n%s", logs)
}
