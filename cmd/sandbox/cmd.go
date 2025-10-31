//go:build linux

package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/grafana/grafana-image-renderer/pkg/sandbox"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel/trace"
	libcap "kernel.org/pub/linux/libs/security/libcap/cap"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:   "_internal_sandbox",
		Usage:  "Starts the browser in a best-effort sandbox.",
		Hidden: true,
		Commands: []*cli.Command{
			{
				Name:  "supported",
				Usage: "Check if the current environment supports sandboxing. This is best-effort.",
				Action: func(ctx context.Context, c *cli.Command) error {
					if !sandbox.Supported(ctx) {
						return fmt.Errorf("sandboxing is not supported in this environment")
					}
					return nil
				},
			},
			{
				Name:  "run",
				Usage: "Run a command inside the sandbox.",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "mount",
						Usage: "Additional mount points to bind into the sandbox in the form of host_path:container_path(:rw)",
						Validator: func(s []string) error {
							for _, s := range s {
								if _, err := parseBindMount(s); err != nil {
									return fmt.Errorf("invalid --mount value %q: %w", s, err)
								}
							}
							return nil
						},
					},
					&cli.StringFlag{
						Name:  "cwd",
						Usage: "The working directory inside the sandbox.",
						Value: "/tmp",
					},
					&cli.StringFlag{
						Name:  "trace",
						Usage: "The OpenTelemetry trace ID to use in logs. This does not change anything about the browser, only the sandbox's log output.",
					},
					&cli.StringFlag{
						Name:     "tmp",
						Usage:    "The directory to mount as /tmp insidee the sandbox. This is intended to be a read-write bind mount.",
						Required: true,
					},
				},
				Action: run,
			},
			{
				Name:            "bootstrap",
				Usage:           "Bootstrap the sandbox environment.",
				SkipFlagParsing: true,
				Action: func(ctx context.Context, c *cli.Command) error {
					newRoot, err := os.MkdirTemp("", "")
					if err != nil {
						return fmt.Errorf("failed to create temp dir: %w", err)
					}
					defer func() { _ = os.RemoveAll(newRoot) }()

					cmd := exec.CommandContext(ctx, "/proc/self/exe", append([]string{"_internal_sandbox", "run"}, c.Args().Slice()...)...)
					cmd.Dir = newRoot
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.SysProcAttr = &syscall.SysProcAttr{
						Pdeathsig:  syscall.SIGKILL,
						Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER,
						UidMappings: []syscall.SysProcIDMap{
							{
								ContainerID: 0,
								HostID:      os.Getuid(),
								Size:        1,
							},
						},
						GidMappings: []syscall.SysProcIDMap{
							{
								ContainerID: 0,
								HostID:      os.Getgid(),
								Size:        1,
							},
						},
					}

					return cmd.Run()
				},
			},
		},
	}
}

func run(ctx context.Context, c *cli.Command) error {
	ctx, err := adoptTrace(ctx, c.String("trace"))
	if err != nil {
		slog.WarnContext(ctx, "failed to adopt trace", "error", err)
	}

	var bindMounts []sandbox.BindMount
	for _, s := range c.StringSlice("mount") {
		bm, err := parseBindMount(s)
		if err != nil {
			// should be unreachable, but easy to just return the error :P
			return fmt.Errorf("invalid --mount value %q: %w", s, err)
		}
		bindMounts = append(bindMounts, bm)
	}
	bindMounts = append(bindMounts, sandbox.BindMount{
		Source:      c.String("tmp"),
		Destination: "/tmp",
		ReadWrite:   true,
	})

	command := c.Args().Slice()
	if len(command) == 0 {
		return fmt.Errorf("no command specified to run in sandbox")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	if err := sandbox.SetupFS(ctx, cwd, bindMounts); err != nil {
		return fmt.Errorf("failed to setup sandbox filesystem: %w", err)
	}

	if err := shedCapabilities(); err != nil {
		slog.WarnContext(ctx, "failed to shed capabilities", "error", err)
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = c.String("cwd")
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL

	// TODO: Ensure this respects signals properly?
	return cmd.Run()
}

func parseBindMount(s string) (sandbox.BindMount, error) {
	host, container, found := strings.Cut(s, ":")
	if !found {
		return sandbox.BindMount{}, fmt.Errorf("invalid mount format, expected host_path:container_path(:rw)")
	}
	container, rw, _ := strings.Cut(container, ":")

	if !filepath.IsAbs(host) {
		return sandbox.BindMount{}, fmt.Errorf("host path must be absolute: %s", host)
	} else if !filepath.IsAbs(container) {
		return sandbox.BindMount{}, fmt.Errorf("container path must be absolute: %s", container)
	} else if rw != "" && rw != "rw" {
		return sandbox.BindMount{}, fmt.Errorf("invalid mount option (must be rw or absent): %s", rw)
	}

	return sandbox.BindMount{
		Source:      host,
		Destination: container,
		ReadWrite:   rw == "rw",
	}, nil
}

func adoptTrace(ctx context.Context, traceID string) (context.Context, error) {
	if traceID == "" {
		return ctx, nil
	}
	tid, err := trace.TraceIDFromHex(traceID)
	if err != nil {
		return ctx, fmt.Errorf("invalid trace ID: %w", err)
	}
	return trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: tid,
		Remote:  true,
	})), nil
}

func shedCapabilities() error {
	if err := libcap.DropBound(libcap.SYS_CHROOT, libcap.SYS_ADMIN, libcap.SETPCAP); err != nil {
		return fmt.Errorf("failed to drop capabilities: %w", err)
	}
	return nil
}
