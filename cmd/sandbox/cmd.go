package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/grafana/grafana-image-renderer/pkg/sandbox"
	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:   "_internal_sandbox",
		Usage:  "Starts the browser in a best-effort sandbox.",
		Hidden: true,
		Action: run,
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
				Name:  "bootstrap",
				Usage: "Bootstrap the sandbox environment.",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "mount",
						Usage: "Additional mount points to bind into the sandbox in the form of host_path:container_path(:ro)",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					newRoot, err := os.MkdirTemp("", "")
					if err != nil {
						return fmt.Errorf("failed to create temp dir: %w", err)
					}
					defer os.RemoveAll(newRoot)

					cmd := exec.CommandContext(ctx, "/proc/self/exe", append([]string{"_internal_sandbox"}, c.Args().Slice()...)...)
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
	command := c.Args().Slice()
	if len(command) == 0 {
		return fmt.Errorf("no command specified to run in sandbox")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	if err := sandbox.SetupFS(ctx, cwd); err != nil {
		return fmt.Errorf("failed to setup sandbox filesystem: %w", err)
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL

	// TODO: Ensure this respects signals properly?
	return cmd.Run()
}
