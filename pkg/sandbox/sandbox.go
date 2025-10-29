package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// Supported attempts to determine if the current file-system will permit using our sandboxing.
//
// Upon any error, it returns false.
func Supported(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// If we can get a sandboxed `true` to return with exit code 0, we're generally pretty good.
	// If we can't, well, we can't.
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "_internal_sandbox", "bootstrap", "--", "true")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	return cmd.Run() == nil
}

// SetupFS sets the file system up for the sandbox, such that we cannot escape the jail.
// This should be called from a process inside a mount namespace, ideally also with a PID namespace.
func SetupFS(ctx context.Context, newRoot string) error {
	if err := mountTmpfs(filepath.Join(newRoot, "tmp")); err != nil {
		return fmt.Errorf("failed to mount tmpfs: %w", err)
	}
	for _, mnt := range []string{"dev", "bin", "usr", "lib", "lib64", "var", "home", "opt", "etc", "proc"} {
		_, err := os.Stat(filepath.Join("/", mnt))
		if errors.Is(err, fs.ErrNotExist) {
			slog.DebugContext(ctx, "bind mount skipped, path does not exist", "path", mnt)
			continue
		} else if errors.Is(err, fs.ErrPermission) {
			slog.DebugContext(ctx, "bind mount skipped, permission denied", "path", mnt)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to stat %q: %w", mnt, err)
		}

		if err := bindMount("/", newRoot, mnt, true); err != nil {
			return fmt.Errorf("failed to bind mount /%s: %w", mnt, err)
		}
	}
	// FIXME: Why can't we mount procfs here? Linux gives us an EPERM. We can, however, bind-mount it.
	//  if err := mountProcfs(filepath.Join(newRoot, "proc")); err != nil {
	//  	return fmt.Errorf("failed to mount procfs: %w", err)
	//  }
	if err := chroot(newRoot); err != nil {
		return fmt.Errorf("failed to chroot to %q: %w", newRoot, err)
	}

	return nil
}

// mountProc mounts a new proc(5) filesystem into the given directory.
// If the directory does not exist, it is created.
func mountProcfs(into string) error {
	if err := os.MkdirAll(into, 0o755); err != nil {
		return fmt.Errorf("failed to create %q: %w", into, err)
	}

	if err := syscall.Mount("proc", into, "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount procfs at %q: %w", into, err)
	}
	return nil
}

// mountTmpfs mounts a new tmpfs filesystem into the given directory.
// If the directory does not exist, it is created.
func mountTmpfs(into string) error {
	if err := os.MkdirAll(into, 0o755); err != nil {
		return fmt.Errorf("failed to create %q: %w", into, err)
	}

	if err := syscall.Mount("tmpfs", into, "tmpfs", 0, ""); err != nil {
		return fmt.Errorf("failed to mount tmpfs at %q: %w", into, err)
	}
	return nil
}

func bindMount(oldRoot, newRoot, path string, readOnly bool) error {
	oldPath := filepath.Join(oldRoot, path)
	newPath := filepath.Join(newRoot, path)

	realOldPath, err := resolveSymlink(oldPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink for %q: %w", oldPath, err)
	}
	realNewPath := filepath.Join(newRoot, strings.TrimPrefix(realOldPath, oldRoot))

	oldStat, err := os.Stat(realOldPath)
	if err != nil {
		return fmt.Errorf("failed to stat %q: %w", realOldPath, err)
	}

	// Ensure target (realNewPath) exists with correct type.
	if oldStat.IsDir() {
		if err := os.MkdirAll(realNewPath, 0o755); err != nil {
			return fmt.Errorf("failed to create dir %q: %w", realNewPath, err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(realNewPath), 0o755); err != nil {
			return fmt.Errorf("failed to create parent dir for %q: %w", realNewPath, err)
		}
		if _, err := os.Stat(realNewPath); os.IsNotExist(err) {
			f, cerr := os.OpenFile(realNewPath, os.O_CREATE|os.O_RDONLY, oldStat.Mode()&0o777)
			if cerr != nil {
				return fmt.Errorf("failed to create file %q: %w", realNewPath, cerr)
			}
			_ = f.Close()
		}
	}

	// Perform bind mount with recursive if directory, and read-only if requested.
	flags := uintptr(syscall.MS_BIND)
	if oldStat.IsDir() {
		flags |= syscall.MS_REC
	}
	if readOnly {
		flags |= syscall.MS_RDONLY
	}
	if err := syscall.Mount(realOldPath, realNewPath, "", flags, ""); err != nil {
		return fmt.Errorf("bind mount %q -> %q failed: %w", realOldPath, realNewPath, err)
	}

	// Recreate symlink if original path was a symlink chain.
	if realNewPath != newPath {
		if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			return fmt.Errorf("failed to create parent dir for symlink %q: %w", newPath, err)
		}
		// Remove any existing entry at newPath.
		if _, err := os.Lstat(newPath); err == nil {
			if err := os.Remove(newPath); err != nil {
				return fmt.Errorf("failed to remove existing %q: %w", newPath, err)
			}
		}
		relTarget, err := filepath.Rel(filepath.Dir(newPath), realNewPath)
		if err != nil {
			return fmt.Errorf("failed to compute relative path from %q to %q: %w", newPath, realNewPath, err)
		}
		if err := os.Symlink(relTarget, newPath); err != nil {
			return fmt.Errorf("failed to create symlink %q -> %q: %w", newPath, relTarget, err)
		}
	}

	return nil
}

func chroot(newRoot string) error {
	// FIXME: Attempt to use pivot_root first instead of chroot.
	//   We don't always have access to pivot_root, so we need the fallback, but it is generally regarded as safer.

	// if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
	// 	return fmt.Errorf("failed to bind mount new root %q onto itself: %w", newRoot, err)
	// }

	// oldRoot := filepath.Join(newRoot, ".old_root")
	// if err := os.MkdirAll(oldRoot, 0o755); err != nil {
	// 	return fmt.Errorf("failed to create old root dir %q: %w", oldRoot, err)
	// }
	// if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
	// 	return fmt.Errorf("pivot_root(%q, %q) failed: %w", newRoot, oldRoot, err)
	// }
	// if err := os.Chdir("/"); err != nil {
	// 	return fmt.Errorf("failed to chdir to new root: %w", err)
	// }

	// if err := syscall.Unmount("/.old_root", syscall.MNT_DETACH); err != nil {
	// 	return fmt.Errorf("failed to unmount old root: %w", err)
	// }
	// if err := os.Remove("/.old_root"); err != nil {
	// 	return fmt.Errorf("failed to remove /.old_root: %w", err)
	// }

	if err := syscall.Chroot(newRoot); err != nil {
		return fmt.Errorf("chroot(%q) failed: %w", newRoot, err)
	}

	// We want to be sure we're not remaining in some directory outside the new root.
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to chdir to new root: %w", err)
	}

	return nil
}

func resolveSymlink(path string) (string, error) {
	for {
		stat, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("failed to stat %q: %w", path, err)
		}
		if stat.Mode()&os.ModeSymlink != os.ModeSymlink {
			return path, nil
		}
		src, err := os.Readlink(path)
		if err != nil {
			return "", fmt.Errorf("failed to readlink %q: %w", path, err)
		}
		path = src
	}
}
