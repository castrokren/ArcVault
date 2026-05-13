package runner

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// RealExecutor runs robocopy on Windows or rsync on Unix/Mac.
// This is the production executor wired into agent/main.go.
func RealExecutor(job Job) (exitCode int, output string) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// robocopy exit codes: 0-7 are success/warning, 8+ are errors
		cmd = exec.Command("robocopy", job.SourcePath, job.DestPath, "/E", "/LOG+:NUL")
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
			// robocopy: codes 1-7 mean success with warnings/copies made
			if runtime.GOOS == "windows" && exitCode <= 7 {
				exitCode = 0
			}
		} else {
			exitCode = 1
			output = fmt.Sprintf("failed to run executor: %v\n%s", err, output)
		}
	}

	return exitCode, output
}
