package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/676
func TestRegression676(t *testing.T) {
	LongTest(t)
	t.Parallel()

	// Exercise the custom CA workflow documented in docs/sources/troubleshooting.md.
	// Merely finding these commands does not detect missing shared libraries.
	const command = `set -e
openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
  -subj '/CN=Acceptance test CA' \
  -keyout /tmp/test-ca.key \
  -out /usr/local/share/ca-certificates/test-ca.crt
update-ca-certificates --fresh
openssl verify /usr/local/share/ca-certificates/test-ca.crt
mkdir -p /tmp/test-ca-nssdb
certutil -N -d sql:/tmp/test-ca-nssdb --empty-password
certutil -A -d sql:/tmp/test-ca-nssdb -n 'Acceptance test CA' -t C \
  -i /usr/local/share/ca-certificates/test-ca.crt
certutil -L -d sql:/tmp/test-ca-nssdb -n 'Acceptance test CA'`
	exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-c"}, []string{command}, WithUser("0"))
	require.Zero(t, exitCode, "custom CA trust setup failed: %s", logs)
	require.Contains(t, logs, "/usr/local/share/ca-certificates/test-ca.crt: OK")
	require.Contains(t, logs, "Acceptance test CA")
}
