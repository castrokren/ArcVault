package runner

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// RealExecutor runs robocopy on Windows or rsync on Unix/Mac, streaming parsed
// progress to report as the backup proceeds. If job.Command is non-empty the
// command is executed directly via the shell with no progress parsing.
//
// This is the production executor wired into agent/main.go.
func RealExecutor(job Job, report ProgressFunc) (exitCode int, output string) {
	if report == nil {
		report = Noop
	}

	// Template-fired job: run the command directly, no progress parsing.
	if job.Command != "" {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", job.Command)
		} else {
			cmd = exec.Command("sh", "-c", job.Command)
		}
		out, err := cmd.CombinedOutput()
		output = string(out)
		exitCode = extractExitCode(err, false)
		report(100, []string{})
		return exitCode, output
	}

	if runtime.GOOS == "windows" {
		return runRobocopy(job, report)
	}
	return runRsync(job, report)
}

// runRobocopy builds and streams a robocopy command.
//
// Flag notes:
//   - /NP is intentionally OMITTED so per-file transfer percentages appear in stdout.
//   - /NFL /NDL suppress the verbose file/dir listing — keeps output focused on % lines.
//   - /E recurse into subdirectories, /R:0 no retries, /W:0 no wait between retries.
func runRobocopy(job Job, report ProgressFunc) (int, string) {
	args := []string{job.SourcePath, job.DestPath, "/E", "/R:0", "/W:0", "/NFL", "/NDL"}
	if job.SyncFlags != nil {
		args = append(args, job.SyncFlags.ToRobocopyArgs()...)
	}
	cmd := exec.Command("robocopy", args...)
	return streamRobocopy(cmd, report)
}

// runRsync builds and streams an rsync command.
//
// --info=progress2 replaces -v and emits a single overall percentage line,
// making it straightforward to parse without per-file noise.
func runRsync(job Job, report ProgressFunc) (int, string) {
	src := strings.TrimRight(job.SourcePath, "/") + "/"
	args := []string{"-a", "--info=progress2", src, job.DestPath}
	if job.SyncFlags != nil {
		args = append(args, job.SyncFlags.ToRsyncArgs()...)
	}
	cmd := exec.Command("rsync", args...)
	return streamRsync(cmd, report)
}

// streamRobocopy starts cmd, splits stdout on \r/\n (robocopy uses bare \r for
// in-place progress), and feeds ParseRobocopyLine results to report.
//
// Stdout and stderr are captured into separate buffers to avoid the data race
// that would occur if both wrote to the same bytes.Buffer concurrently
// (cmd.Stderr is drained by an internal goroutine while we scan stdout).
func streamRobocopy(cmd *exec.Cmd, report ProgressFunc) (int, string) {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		out, _ := cmd.CombinedOutput()
		return 1, string(out)
	}
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return 1, fmt.Sprintf("failed to start robocopy: %v", err)
	}

	filesDone := 0
	var pending []string

	scanner := bufio.NewScanner(io.TeeReader(stdoutPipe, &stdoutBuf))
	scanner.Buffer(make([]byte, 256*1024), 256*1024) // 256 KB — robocopy lines are wide
	scanner.Split(SplitOnCRLF)

	for scanner.Scan() {
		line := scanner.Text()
		if pct, complete, ok := ParseRobocopyLine(line); ok {
			if complete {
				filesDone++
				pending = append(pending, fmt.Sprintf("file %d complete", filesDone))
			}
			report(pct, pending)
			pending = nil
		}
	}

	code := waitCode(cmd, true /* isRobocopy */)
	report(100, pending) // flush any trailing log lines + mark done

	output := stdoutBuf.String()
	if s := stderrBuf.String(); s != "" {
		if output != "" {
			output += "\n"
		}
		output += s
	}
	return code, output
}

// streamRsync starts cmd, scans stdout line by line (splitting on \r/\n for
// robustness), and feeds ParseRsyncLine results to report.
//
// Stdout and stderr use separate buffers to avoid concurrent write races.
func streamRsync(cmd *exec.Cmd, report ProgressFunc) (int, string) {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		out, _ := cmd.CombinedOutput()
		return 1, string(out)
	}
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return 1, fmt.Sprintf("failed to start rsync: %v", err)
	}

	var pending []string
	scanner := bufio.NewScanner(io.TeeReader(stdoutPipe, &stdoutBuf))
	scanner.Buffer(make([]byte, 256*1024), 256*1024)
	scanner.Split(SplitOnCRLF)

	for scanner.Scan() {
		line := scanner.Text()
		if pct, ok := ParseRsyncLine(line); ok {
			report(pct, pending)
			pending = nil
		} else if t := strings.TrimSpace(line); t != "" {
			pending = append(pending, t)
		}
	}

	code := waitCode(cmd, false)
	report(100, pending)

	output := stdoutBuf.String()
	if s := stderrBuf.String(); s != "" {
		if output != "" {
			output += "\n"
		}
		output += s
	}
	return code, output
}

// waitCode calls cmd.Wait and returns the process exit code.
//
// Robocopy exit codes are a bitmask:
//   - Bits 0–2 (codes 1–7):  informational only (files copied, extras, mismatches)
//   - Bit 3 (codes 8–15):    some copy errors occurred but partial copies succeeded
//   - Code 16+:              fatal error (wrong path, access denied entirely)
//
// Codes 1–15 are all normalised to 0 so the job is not marked "failed" for
// partial copies or per-file errors. Code 16+ is a genuine failure.
func waitCode(cmd *exec.Cmd, isRobocopy bool) int {
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if isRobocopy && code >= 1 && code <= 15 {
				return 0
			}
			return code
		}
		return 1
	}
	return 0
}

// extractExitCode pulls the exit code out of a cmd.Run / CombinedOutput error.
// Matches waitCode's robocopy normalisation: codes 1–15 → 0, code 16+ = fatal.
func extractExitCode(err error, isRobocopy bool) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		if isRobocopy && code >= 1 && code <= 15 {
			return 0
		}
		return code
	}
	return 1
}
