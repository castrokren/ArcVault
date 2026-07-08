package runner

import (
	"strings"
)

// AllowedPrograms is a whitelist of programs that are permitted to execute in Phase 2B.
// In Phase 2A (audit mode), all commands execute but are logged as allowed/disallowed.
var AllowedPrograms = map[string]bool{
	"rsync":    true,
	"robocopy": true,
}

// IsWhitelisted returns true if the program name is in the allowed whitelist.
func IsWhitelisted(programName string) bool {
	return AllowedPrograms[strings.ToLower(programName)]
}

// ExtractProgramName extracts the program name from a command string.
// It parses the command arguments respecting quotes and returns the base name
// of the first argument (program path).
//
// Examples:
//
//	"rsync -a src/ dest/" → "rsync"
//	"/usr/bin/rsync -a src/ dest/" → "rsync"
//	"C:\\Program Files\\robocopy src dest" → "robocopy"
//	"/bin/bash -c 'echo hello'" → "bash"
func ExtractProgramName(command string) string {
	args := parseCommandArgs(command)
	if len(args) == 0 {
		return ""
	}

	programPath := args[0]

	// Handle both Unix (/path/to/prog) and Windows (C:\path\to\prog) paths
	// First try to find last backslash (Windows path)
	if idx := strings.LastIndex(programPath, "\\"); idx != -1 {
		return programPath[idx+1:]
	}

	// Then try to find last forward slash (Unix path)
	if idx := strings.LastIndex(programPath, "/"); idx != -1 {
		return programPath[idx+1:]
	}

	// No path separator, return as-is
	return programPath
}
