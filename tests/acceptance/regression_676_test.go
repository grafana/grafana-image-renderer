package acceptance

import (
	"io"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/676
func TestRegression676(t *testing.T) {
	LongTest(t)
	t.Parallel()

	req := testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:      GetDockerImage(t),
			Entrypoint: []string{"tini", "--", "sleep"},
			Cmd:        []string{"120"},
			ConfigModifier: func(c *container.Config) {
				c.User = "0"
			},
		},
	}
	instance, err := testcontainers.GenericContainer(t.Context(), req)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, instance)

	// Exercise the custom-CA workflow without a shell, including hashed trust
	// lookup and NSS import. Merely finding the tools misses broken script hooks.
	for _, cmd := range [][]string{
		{"mkdir", "-p", "/usr/local/share/ca-certificates"},
		{"openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-keyout", "/tmp/test-ca.key", "-out", "/usr/local/share/ca-certificates/test-ca.crt", "-subj", "/CN=Renderer acceptance test CA", "-days", "1"},
		{"update-ca-certificates"},
		{"openssl", "verify", "-no-CAfile", "-no-CAstore", "-CApath", "/etc/ssl/certs", "/usr/local/share/ca-certificates/test-ca.crt"},
		{"certutil", "-N", "--empty-password", "-d", "sql:/tmp"},
		{"certutil", "-A", "-n", "test-ca", "-t", "C,,", "-d", "sql:/tmp", "-i", "/usr/local/share/ca-certificates/test-ca.crt"},
		{"certutil", "-L", "-n", "test-ca", "-d", "sql:/tmp"},
	} {
		exitCode, output, err := instance.Exec(t.Context(), cmd, tcexec.Multiplexed())
		require.NoError(t, err, "could not execute %q", cmd)
		logs, err := io.ReadAll(output)
		require.NoError(t, err)
		require.Zero(t, exitCode, "%q failed: %s", cmd, logs)
	}
}
