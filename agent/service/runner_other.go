//go:build !windows

package service

// RunService on non-Windows systems calls fn directly.
// systemd/launchd manage the process lifecycle externally.
func RunService(fn func()) error {
	fn()
	return nil
}
