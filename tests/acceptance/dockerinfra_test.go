package acceptance

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
)

func LongTest(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping long test in short mode")
	}
}

func GetDockerImage(tb testing.TB) string {
	tb.Helper()
	img := os.Getenv("IMAGE")
	if img == "" {
		if os.Getenv("REQUIRE_ACCEPTANCE") != "" {
			tb.Fatal("IMAGE environment variable not set, cannot run acceptance test")
		} else {
			tb.Skip("IMAGE environment variable not set, skipping acceptance test")
		}
	}
	return img
}

type ContainerOption func(testing.TB, *testcontainers.GenericContainerRequest)

func WithConfigModifier(f func(*container.Config)) ContainerOption {
	return func(tb testing.TB, gcr *testcontainers.GenericContainerRequest) {
		tb.Helper()
		old := gcr.ConfigModifier
		gcr.ConfigModifier = func(c *container.Config) {
			if old != nil {
				old(c)
			}
			f(c)
		}
	}
}

func WithUser(user string) ContainerOption {
	return WithConfigModifier(func(c *container.Config) {
		c.User = user
	})
}

func WithEnv(k, v string) ContainerOption {
	return func(tb testing.TB, gcr *testcontainers.GenericContainerRequest) {
		tb.Helper()
		require.NotEmpty(tb, k, "env key (with value %q) cannot be empty", v)
		if gcr.Env == nil {
			gcr.Env = make(map[string]string)
		}
		gcr.Env[k] = v
	}
}

func WithNetwork(net *testcontainers.DockerNetwork, netAlias string) ContainerOption {
	return func(tb testing.TB, gcr *testcontainers.GenericContainerRequest) {
		tb.Helper()
		gcr.Networks = append(gcr.Networks, net.Name)
		gcr.NetworkAliases = map[string][]string{
			net.Name: {netAlias},
		}
	}
}

func WithArgs(args ...string) ContainerOption {
	return WithConfigModifier(func(c *container.Config) {
		c.Cmd = args
	})
}

type ImageRenderer struct {
	testcontainers.Container
	HTTPEndpoint string
}

func StartImageRenderer(tb testing.TB, options ...ContainerOption) *ImageRenderer {
	tb.Helper()

	httpPort, err := nat.NewPort("tcp", "8081")
	require.NoError(tb, err, "could not construct a TCP port for 8081")

	req := testcontainers.GenericContainerRequest{
		Logger:  log.TestLogger(tb),
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			// TODO: Use Dockerfile instead?
			Image:        GetDockerImage(tb),
			WaitingFor:   wait.ForHealthCheck().WithPollInterval(time.Millisecond * 50),
			ExposedPorts: []string{"8081/tcp"},
			ConfigModifier: func(c *container.Config) {
				if c.Healthcheck == nil {
					c.Healthcheck = &container.HealthConfig{}
				}
				c.Healthcheck.StartInterval = time.Millisecond * 50
				c.Healthcheck.StartPeriod = time.Second * 5
			},
		},
	}
	for _, f := range options {
		f(tb, &req)
	}

	container, err := testcontainers.GenericContainer(tb.Context(), req)
	require.NoError(tb, err, "could not start service container?")
	testcontainers.CleanupContainer(tb, container)

	endpoint, err := container.PortEndpoint(tb.Context(), httpPort, "http")
	require.NoError(tb, err, "could not get HTTP endpoint of container")

	return &ImageRenderer{
		Container:    container,
		HTTPEndpoint: endpoint,
	}
}

// Run the given command on the image to completion.
func RunImageRendererWithCommand(tb testing.TB, entrypoint []string, cmd []string, options ...ContainerOption) (int, string) {
	tb.Helper()

	req := testcontainers.GenericContainerRequest{
		Logger:  log.TestLogger(tb),
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			// TODO: Use Dockerfile instead?
			Image:      GetDockerImage(tb),
			Entrypoint: entrypoint,
			Cmd:        cmd,
			WaitingFor: wait.ForExit(),
		},
	}
	for _, f := range options {
		f(tb, &req)
	}

	container, err := testcontainers.GenericContainer(tb.Context(), req)
	require.NoError(tb, err, "could not start container")

	state, err := container.State(tb.Context())
	require.NoError(tb, err, "could not get container state")

	logs, err := container.Logs(tb.Context())
	require.NoError(tb, err, "could not get container logs")
	defer logs.Close()
	contents, err := io.ReadAll(logs)
	require.NoError(tb, err, "could not read container logs")

	return state.ExitCode, strings.TrimSpace(string(contents))
}

type Grafana struct {
	testcontainers.Container
	HTTPEndpoint string
}

func StartGrafana(tb testing.TB, options ...ContainerOption) *Grafana {
	tb.Helper()

	httpPort, err := nat.NewPort("tcp", "3000")
	require.NoError(tb, err, "could not construct a TCP port for 3000")

	req := testcontainers.GenericContainerRequest{
		Logger:  log.TestLogger(tb),
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "docker.io/grafana/grafana-enterprise:main",
			WaitingFor:   wait.ForHTTP("/healthz").WithPort(httpPort).WithAllowInsecure(true),
			ExposedPorts: []string{"3000/tcp"},
		},
	}
	for _, f := range options {
		f(tb, &req)
	}

	container, err := testcontainers.GenericContainer(tb.Context(), req)
	require.NoError(tb, err, "could not start service container?")
	testcontainers.CleanupContainer(tb, container)

	endpoint, err := container.PortEndpoint(tb.Context(), httpPort, "http")
	require.NoError(tb, err, "could not get HTTP endpoint of container")

	return &Grafana{
		Container:    container,
		HTTPEndpoint: endpoint,
	}
}
