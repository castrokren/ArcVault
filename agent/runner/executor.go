package runner

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// RealExecutor runs robocopy on Windows or rsync on Unix/Mac.
// If job.Command is non-empty, it is executed directly via the shell instead
// of building a robocopy/rsync command from source_path/dest_path.
// This is the production executor wired into agent/main.go.
func RealExecutor(job Job) (exitCode int, output string) {
	var cmd *exec.Cmd

	if job.Command != "" {
		// Template-fired job: run the command directly.
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", job.Command)
		} else {
			cmd = exec.Command("sh", "-c", job.Command)
		}
	} else if runtime.GOOS == "windows" {
		// robocopy exit codes: 0-7 are success/warning, 8+ are errors
		// Flags: /E (recurse), /R:0 (no retries), /W:0 (no wait), /NP (no progress), /NFL /NDL (suppress logs)
		cmd = exec.Command("robocopy", job.SourcePath, job.DestPath, "/E", "/R:0", "/W:0", "/NP", "/NFL", "/NDL")
	} else {
		// rsync: -a archive, -v verbose, trailing slash copies contents
		src := strings.TrimRight(job.SourcePath, "/") + "/"
		cmd = exec.Command("rsync", "-av", src, job.DestPath)
	}

	out, err := cmd.CombinedOutput()
	output = string(out)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			// robocopy: codes 1-7 mean success with warnings/copies made.
			// Only apply this for non-command jobs (robocopy path).
			if runtime.GOOS == "windows" && job.Command == "" && exitCode <= 7 {
				exitCode = 0
			}
		} else {
			exitCode = 1
			output = fmt.Sprintf("failed to run executor: %v\n%s", err, output)
		}
	}

	return exitCode, output
}
