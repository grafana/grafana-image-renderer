//go:build !linux

package sandbox

import "context"

// Supported attempts to determine if the current file-system will permit using our sandboxing.
//
// Upon any error, it returns false.
func Supported(_ context.Context) bool {
	return false
}
