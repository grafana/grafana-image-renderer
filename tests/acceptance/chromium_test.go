package acceptance

import (
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestChromiumInstalled(t *testing.T) {
	LongTest(t)
	t.Parallel()

	exitCode, logs := RunImageRendererWithCommand(t, []string{"chromium", "--version"}, nil)
	require.Zero(t, exitCode, "chromium did not exist in the container (--version failed)")
	require.Regexp(t, `^Chromium [0-9]+`, logs, "chromium --version did not return expected output")
}

func TestSandboxInheritsDebuggingPipes(t *testing.T) {
	LongTest(t)
	t.Parallel()

	// Send a request through inherited fd 3, across both sandbox launches,
	// then return its response through inherited fd 4.
	const script = `
mkdir -p /tmp/gir-pipe-test
exec 3< <(printf 'pipe-request\n')
exec 4>&1
exec /usr/bin/grafana-image-renderer _internal_sandbox bootstrap --tmp /tmp/gir-pipe-test -- /bin/sh -c 'IFS= read -r message <&3; printf "received:%s\n" "$message" >&4' --remote-debugging-pipe
`
	exitCode, logs := RunImageRendererWithCommand(t, []string{"/bin/bash", "-c"}, []string{script},
		func(_ testing.TB, req *testcontainers.GenericContainerRequest) {
			req.HostConfigModifier = func(hc *container.HostConfig) {
				hc.CapAdd = append(hc.CapAdd, "CAP_SYS_ADMIN", "CAP_SYS_CHROOT")
				hc.SecurityOpt = append(hc.SecurityOpt, "apparmor=unconfined")
			}
		})
	require.Zero(t, exitCode, "sandbox failed: %s", logs)
	require.Contains(t, logs, "received:pipe-request")
}
