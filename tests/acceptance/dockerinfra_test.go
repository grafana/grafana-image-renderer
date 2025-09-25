package acceptance

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
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

func OnlyEnterprise(tb testing.TB) {
	tb.Helper()
	jwt := licensePath()
	if _, err := os.Stat(jwt); os.IsNotExist(err) {
		if os.Getenv("REQUIRE_ENTERPRISE") == "true" {
			tb.Fatalf("enterprise license file %q does not exist, cannot run enterprise-only test", jwt)
		} else {
			tb.Skipf("enterprise license file %q does not exist, skipping enterprise-only test", jwt)
		}
	}
}

func licensePath() string {
	if p := os.Getenv("LICENSE_JWT"); p != "" {
		return p
	}
	return path.Join(findGitRoot(), "license.jwt")
}

func findGitRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}

	dir := wd
	for {
		if _, err := os.Stat(path.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := path.Dir(dir)
		if parent == "." || parent == dir {
			return wd
		}
		dir = parent
	}
}

func GetDockerImage(tb testing.TB) string {
	tb.Helper()
	img := os.Getenv("IMAGE")
	if img == "" {
		if os.Getenv("REQUIRE_ACCEPTANCE") == "true" {
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
			LogConsumerCfg: containerLogs(tb, "image-renderer"),
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
			Image:          GetDockerImage(tb),
			Entrypoint:     entrypoint,
			Cmd:            cmd,
			WaitingFor:     wait.ForExit(),
			LogConsumerCfg: containerLogs(tb, "image-renderer"),
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
	defer func() { _ = logs.Close() }()
	contents, err := io.ReadAll(logs)
	require.NoError(tb, err, "could not read container logs")

	return state.ExitCode, strings.TrimSpace(string(contents))
}

type Grafana struct {
	testcontainers.Container
	HTTPEndpoint string
}

//go:embed fixtures
var fixturesFS embed.FS

func StartGrafana(tb testing.TB, options ...ContainerOption) *Grafana {
	tb.Helper()

	httpPort, err := nat.NewPort("tcp", "3000")
	require.NoError(tb, err, "could not construct a TCP port for 3000")

	req := testcontainers.GenericContainerRequest{
		Logger:  log.TestLogger(tb),
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "docker.io/grafana/grafana-enterprise:12.1.1",
			Env: map[string]string{
				"GF_FEATURE_TOGGLES_ENABLE":  "renderAuthJWT",
				"GF_LOG_FILTERS":             "debug",
				"GF_ENTERPRISE_LICENSE_PATH": "/license.jwt",
			},
			WaitingFor: wait.ForAll(
				wait.ForHTTP("/healthz").WithPort(httpPort).WithAllowInsecure(true),
				wait.ForLog("inserting datasource from configuration"), // from the provisioning files we add
				wait.ForLog("finished to provision dashboards"),        // from the provisioning files we add
			),
			ExposedPorts:   []string{"3000/tcp"},
			Files:          createGrafanaProvisioningFiles(tb),
			LogConsumerCfg: containerLogs(tb, "grafana"),
		},
	}
	if _, err := os.Stat(licensePath()); err == nil {
		req.Files = append(req.Files, testcontainers.ContainerFile{
			HostFilePath:      licensePath(),
			ContainerFilePath: "/license.jwt",
			FileMode:          0o777,
		})
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

func createGrafanaProvisioningFiles(tb testing.TB) []testcontainers.ContainerFile {
	tb.Helper()

	files, err := fsToContainerFiles(embedFSSub(tb, fixturesFS, "fixtures/dashboards"), ".", "/usr/share/grafana/dashboards")
	require.NoError(tb, err, "could not create container files from embedded dashboards")

	provisioningFiles, err := fsToContainerFiles(embedFSSub(tb, fixturesFS, "fixtures/provisioning"), ".", "/etc/grafana/provisioning")
	require.NoError(tb, err, "could not create container files from embedded provisioning datasources")
	files = append(files, provisioningFiles...)

	return files
}

type embedFS interface {
	fs.FS
	fs.ReadDirFS
	fs.ReadFileFS
}

func embedFSSub(tb testing.TB, f embedFS, dir string) embedFS {
	sub, err := fs.Sub(f, dir)
	require.NoError(tb, err, "could not get sub fs for dir %q", dir)
	if efs, ok := sub.(embedFS); ok {
		return efs
	}
	require.Fail(tb, "sub fs is not an embedFS", "got type %T", sub)
	panic("unreachable")
}

func fsToContainerFiles(fs embedFS, baseSrc, baseDst string) ([]testcontainers.ContainerFile, error) {
	var files []testcontainers.ContainerFile

	entries, err := fs.ReadDir(baseSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded dir %q: %w", baseSrc, err)
	}
	for _, f := range entries {
		if f.IsDir() {
			subFiles, err := fsToContainerFiles(fs, path.Join(baseSrc, f.Name()), path.Join(baseDst, f.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to join with subdir %q: %w", f.Name(), err)
			}
			files = append(files, subFiles...)
			continue
		}

		file, err := fs.Open(path.Join(baseSrc, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", path.Join(baseSrc, f.Name()), err)
		}

		files = append(files, testcontainers.ContainerFile{
			Reader:            file,
			ContainerFilePath: path.Join(baseDst, f.Name()),
			FileMode:          0o777,
		})
	}

	return files, nil
}

func StartPrometheus(tb testing.TB, options ...ContainerOption) {
	tb.Helper()

	req := testcontainers.GenericContainerRequest{
		Logger:  log.TestLogger(tb),
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "prom/prometheus:latest",
			WaitingFor: wait.ForAll(
				wait.ForHTTP("/-/healthy"),
				wait.ForLog("Server is ready"),
			),
			LogConsumerCfg: containerLogs(tb, "prometheus"),
		},
	}
	for _, f := range options {
		f(tb, &req)
	}

	container, err := testcontainers.GenericContainer(tb.Context(), req)
	require.NoError(tb, err, "could not start service container?")
	testcontainers.CleanupContainer(tb, container)
}

func containerLogs(tb testing.TB, container string) *testcontainers.LogConsumerConfig {
	tb.Helper()
	return &testcontainers.LogConsumerConfig{
		Consumers: []testcontainers.LogConsumer{
			&testingLogConsumer{name: container, test: tb.Name()},
		},
	}
}

type testingLogConsumer struct {
	name string
	test string
}

func (t *testingLogConsumer) Accept(l testcontainers.Log) {
	txt := string(l.Content)
	txt = strings.TrimSpace(txt)
	fmt.Printf("[%s|%s] %s\n", t.test, t.name, txt)
}
