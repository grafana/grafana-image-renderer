//go:build !linux

package sandbox

import (
	"context"
	"os/exec"
)

// SetParentDeathSignal has no effect on platforms other than Linux.
func SetParentDeathSignal(_ *exec.Cmd) {}

// Supported attempts to determine if the current file-system will permit using our sandboxing.
//
// Upon any error, it returns false.
func Supported(_ context.Context) bool {
	return false
}
