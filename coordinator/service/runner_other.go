//go:build !windows

package service

import (
	"io/fs"

	"arcvault/coordinator/cmd"
	"arcvault/coordinator/config"
)

// RunService on non-Windows platforms starts the server directly,
// since systemd/launchd handle the process lifecycle externally.
func RunService(cfg *config.Config, staticFS fs.FS) error {
	return cmd.StartCommand(cfg, staticFS)
}
