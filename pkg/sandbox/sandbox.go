//go:build linux

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
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "_internal_sandbox", "bootstrap", "--tmp", "/tmp", "--", "true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = nil
	return cmd.Run() == nil
}

// BindMount defines a mount that should exist in the sandbox.
type BindMount struct {
	// Source is the absolute path on the host to bind-mount into the sandbox.
	Source string
	// Destination is the absolute path in the sandbox where the source is mounted.
	// This should _not_ include the new root prefix; if you want `/a` to be mounted at `/a` (inside the sandbox), just assign this to `/a`, not `/new-root/a`.
	Destination string
	// ReadWrite determines whether the mount is read-write or read-only (default).
	ReadWrite bool
}

// SetupFS sets the file system up for the sandbox, such that we cannot escape the jail.
// This should be called from a process inside a mount namespace, ideally also with a PID namespace.
func SetupFS(ctx context.Context, newRoot string, bindMounts []BindMount) error {
	mountedProc := false
	if !anyMount(bindMounts, "/proc") {
		if err := mountProcfs(filepath.Join(newRoot, "proc")); err != nil {
			if errors.Is(err, syscall.EPERM) || errors.Is(err, os.ErrPermission) {
				slog.WarnContext(ctx, "mounting new procfs not permitted, will attempt to bind-mount existing /proc")
			} else {
				return fmt.Errorf("failed to mount procfs: %w", err)
			}
		} else {
			mountedProc = true
		}
	}
	defaultMounts := []string{"bin", "usr", "lib", "lib64", "var", "home", "opt", "etc", "dev"}
	if !mountedProc {
		defaultMounts = append(defaultMounts, "proc")
	}
	for _, mnt := range defaultMounts {
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

		if err := replicateBaseBindMount("/", newRoot, mnt, true); err != nil {
			return fmt.Errorf("failed to bind mount /%s: %w", mnt, err)
		}
	}
	for _, bm := range bindMounts {
		if err := bm.mount("/", newRoot); err != nil {
			return fmt.Errorf("failed to apply bind mount %q -> %q: %w", bm.Source, bm.Destination, err)
		}
	}
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

func (b BindMount) mount(oldRoot, newRoot string) error {
	oldPath := filepath.Join(oldRoot, b.Source)
	newPath := filepath.Join(newRoot, b.Destination)

	// There is no symlink recreation in these cases. Here, we expect the input to have all the symlink work done for us.
	// We can revisit this later if we have to.

	realOldPath, err := resolveSymlink(oldPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink for %q: %w", oldPath, err)
	}

	oldStat, err := os.Stat(realOldPath)
	if err != nil {
		return fmt.Errorf("failed to stat %q: %w", realOldPath, err)
	}

	// We want to be sure the right type of file descriptor exists.
	if oldStat.IsDir() {
		if err := os.MkdirAll(newPath, 0o755); err != nil {
			return fmt.Errorf("failed to create dir %q: %w", newPath, err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			return fmt.Errorf("failed to create parent dir for %q: %w", newPath, err)
		}
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			f, cerr := os.OpenFile(newPath, os.O_CREATE|os.O_RDONLY, oldStat.Mode()&0o777)
			if cerr != nil {
				return fmt.Errorf("failed to create file %q: %w", newPath, cerr)
			}
			_ = f.Close()
		}
	}

	flags := uintptr(syscall.MS_BIND)
	if oldStat.IsDir() {
		flags |= syscall.MS_REC
	}
	if !b.ReadWrite {
		flags |= syscall.MS_RDONLY
	}
	if err := syscall.Mount(realOldPath, newPath, "", flags, ""); err != nil {
		return fmt.Errorf("bind mount %q -> %q failed: %w", realOldPath, newPath, err)
	}

	return nil
}

func anyMount(bindMounts []BindMount, path string) bool {
	for _, bm := range bindMounts {
		if filepath.Clean(bm.Destination) == filepath.Clean(path) {
			return true
		}
	}
	return false
}

func replicateBaseBindMount(oldRoot, newRoot, path string, readOnly bool) error {
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
	useChroot := func() error {
		if err := syscall.Chroot(newRoot); err != nil {
			return fmt.Errorf("chroot(%q) failed: %w", newRoot, err)
		}

		// We want to be sure we're not remaining in some directory outside the new root.
		if err := os.Chdir("/"); err != nil {
			return fmt.Errorf("failed to chdir to new root: %w", err)
		}

		return nil
	}

	// We will first try to pivot_root. If we succeed with the pivot_root call, we're past the point of no return.
	// Until then, we can still undo our work and try a chroot instead.
	// pivot_root is significantly safer than chroot, so we prefer it when available.
	// That said, we should still be relatively fine with chroot...

	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		if cherr := useChroot(); cherr != nil {
			return fmt.Errorf("failed to bind mount new root %q onto itself: %v; additionally, chroot fallback failed: %w", newRoot, err, cherr)
		} else {
			return nil // we chrooted successfully instead
		}
	}

	oldRoot := filepath.Join(newRoot, ".old_root")
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		if umntErr := syscall.Unmount(newRoot, syscall.MNT_DETACH); umntErr != nil {
			return fmt.Errorf("failed to unmount new root after mkdir failure: %v; original mkdir error: %w", umntErr, err)
		}
		if cherr := useChroot(); cherr != nil {
			return fmt.Errorf("failed to bind mount new root %q onto itself: %v; additionally, chroot fallback failed: %w", newRoot, err, cherr)
		} else {
			return nil // we chrooted successfully instead
		}
	}
	if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
		// We can leave the directory be. It is empty, so we don't particularly care.
		if umntErr := syscall.Unmount(newRoot, syscall.MNT_DETACH); umntErr != nil {
			return fmt.Errorf("failed to unmount new root after pivot_root failure: %v; original pivot_root error: %w", umntErr, err)
		}
		if cherr := useChroot(); cherr != nil {
			return fmt.Errorf("failed to pivot_root to %q: %v; additionally, chroot fallback failed: %w", newRoot, err, cherr)
		} else {
			return nil // we chrooted successfully instead
		}
	}
	// We're past the point of no return now.
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to chdir to new root: %w", err)
	}

	if err := syscall.Unmount("/.old_root", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to unmount old root: %w", err)
	}
	if err := os.Remove("/.old_root"); err != nil {
		return fmt.Errorf("failed to remove /.old_root: %w", err)
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
