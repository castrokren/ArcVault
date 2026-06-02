package runner

import (
	"fmt"
)

// Unit conversion constants
const (
	secondsPerDay = 86400      // 24 * 60 * 60
	bytesPerMB    = 1024 * 1024
)

// SyncFlags holds advanced sync options for robocopy and rsync.
type SyncFlags struct {
	// Mirror enables mirror mode (delete destination files not in source)
	Mirror bool `json:"mirror"`

	// MaxAge is the maximum age of files to sync (in days)
	MaxAge int `json:"max_age,omitempty"`

	// MinAge is the minimum age of files to sync (in days)
	MinAge int `json:"min_age,omitempty"`

	// MaxSize is the maximum file size to sync (in MB)
	MaxSize int `json:"max_size,omitempty"`

	// ExcludeFiles is a list of file patterns to exclude (e.g., "*.tmp", "*.log")
	ExcludeFiles []string `json:"exclude_files,omitempty"`

	// ExcludeDirs is a list of directory patterns to exclude (e.g., ".git", "node_modules")
	ExcludeDirs []string `json:"exclude_dirs,omitempty"`
}

// appendExcludePatterns adds exclude patterns to an args slice for rsync.
// Both ExcludeFiles and ExcludeDirs use the same --exclude flag.
func (sf *SyncFlags) appendExcludePatterns(args []string) []string {
	for _, pattern := range sf.ExcludeFiles {
		args = append(args, fmt.Sprintf("--exclude=%s", pattern))
	}
	for _, pattern := range sf.ExcludeDirs {
		args = append(args, fmt.Sprintf("--exclude=%s", pattern))
	}
	return args
}

// Validate checks if SyncFlags values are valid.
func (sf *SyncFlags) Validate() error {
	if sf.MaxAge < 0 {
		return fmt.Errorf("MaxAge cannot be negative: %d", sf.MaxAge)
	}
	if sf.MinAge < 0 {
		return fmt.Errorf("MinAge cannot be negative: %d", sf.MinAge)
	}
	if sf.MaxSize < 0 {
		return fmt.Errorf("MaxSize cannot be negative: %d", sf.MaxSize)
	}
	if sf.MinAge > 0 && sf.MaxAge > 0 && sf.MinAge > sf.MaxAge {
		return fmt.Errorf("MinAge (%d) cannot be greater than MaxAge (%d)", sf.MinAge, sf.MaxAge)
	}
	return nil
}

// ToRobocopyArgs converts SyncFlags to robocopy command-line arguments.
// Returns a slice of arguments that can be passed to robocopy.exe.
func (sf *SyncFlags) ToRobocopyArgs() []string {
	var args []string

	if sf.Mirror {
		args = append(args, "/MIR")
	}

	if sf.MaxAge > 0 {
		args = append(args, fmt.Sprintf("/MAXAGE:%d", sf.MaxAge))
	}

	if sf.MinAge > 0 {
		args = append(args, fmt.Sprintf("/MINAGE:%d", sf.MinAge))
	}

	if sf.MaxSize > 0 {
		args = append(args, fmt.Sprintf("/MAXSIZE:%dM", sf.MaxSize))
	}

	if len(sf.ExcludeFiles) > 0 {
		args = append(args, "/XF")
		args = append(args, sf.ExcludeFiles...)
	}

	if len(sf.ExcludeDirs) > 0 {
		args = append(args, "/XD")
		args = append(args, sf.ExcludeDirs...)
	}

	return args
}

// ToRsyncArgs converts SyncFlags to rsync command-line arguments.
// Returns a slice of arguments that can be passed to rsync.
func (sf *SyncFlags) ToRsyncArgs() []string {
	var args []string

	if sf.Mirror {
		args = append(args, "--delete")
	}

	if sf.MaxAge > 0 {
		// rsync uses seconds; convert days to seconds
		seconds := sf.MaxAge * secondsPerDay
		args = append(args, fmt.Sprintf("--max-age=%d", seconds))
	}

	if sf.MinAge > 0 {
		// rsync uses seconds; convert days to seconds
		seconds := sf.MinAge * secondsPerDay
		args = append(args, fmt.Sprintf("--min-age=%d", seconds))
	}

	if sf.MaxSize > 0 {
		// rsync uses bytes; convert MB to bytes
		bytes := sf.MaxSize * bytesPerMB
		args = append(args, fmt.Sprintf("--maxsize=%d", bytes))
	}

	// rsync exclude patterns (file and directory patterns use same flag)
	args = sf.appendExcludePatterns(args)

	return args
}
