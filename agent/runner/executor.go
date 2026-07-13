package runner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"unicode"
)

// CommandAuditor is a callback function signature for logging command execution in audit mode.
// It receives the execution context and logs it to a persistent audit store.
type CommandAuditor func(ctx CommandAuditContext)

// CommandAuditContext holds audit information for a command execution.
type CommandAuditContext struct {
	TemplateID    *string
	JobID         *string
	CommandString string
	ProgramName   string
	IsWhitelisted bool
	Mode          string // "audit" or "enforce"
	AuditResult   string
	AgentID       string
}

// parseCommandArgs splits a command string into arguments, respecting quoted strings.
// This avoids shell interpretation while still allowing multi-word arguments.
// Handles both single and double quotes, properly stripping them from the result.
// Format: "program arg1 arg2 'arg with spaces' \"another quoted arg\""
// Windows paths with backslashes are preserved correctly within quotes.
func parseCommandArgs(cmdStr string) []string {
	var args []string
	var current strings.Builder
	var inSingle, inDouble bool
	var i int

	for i < len(cmdStr) {
		ch := rune(cmdStr[i])

		// Handle single quotes: toggle state, don't add to output
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			i++
			continue
		}

		// Handle double quotes: toggle state, don't add to output
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			i++
			continue
		}

		// Handle whitespace as delimiter (when not quoted)
		if unicode.IsSpace(ch) && !inSingle && !inDouble {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			i++
			continue
		}

		// Add character to current argument (including backslashes in quoted strings)
		current.WriteRune(ch)
		i++
	}

	// Add final argument if any
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// RealExecutor runs robocopy on Windows or rsync on Unix/Mac, streaming parsed
// progress to report as the backup proceeds. If job.Command is non-empty the
// command is executed via explicit arguments (not via shell) for security.
//
// This is the production executor wired into agent/main.go.
func RealExecutor(ctx context.Context, job Job, report ProgressFunc) (exitCode int, output string) {
	if report == nil {
		report = Noop
	}

	// Template-fired job: parse command into arguments and execute without shell.
	// Commands are structured as "program arg1 arg2 ..." to avoid shell injection.
	if job.Command != "" {
		args := parseCommandArgs(job.Command)
		if len(args) == 0 {
			return 1, "command is empty after parsing"
		}

		programName := ExtractProgramName(job.Command)
		if !IsWhitelisted(programName) {
			return 1, fmt.Sprintf("command not allowed: %s", programName)
		}

		// Execute the program with parsed arguments, avoiding shell interpretation
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		output = string(out)
		exitCode = extractExitCode(err, false)
		report(100, []string{})
		return exitCode, output
	}

	if runtime.GOOS == "windows" {
		return runRobocopy(ctx, job, report)
	}
	return runRsync(ctx, job, report)
}

// runRobocopy builds and streams a robocopy command.
//
// Flag notes:
//   - /NP is intentionally OMITTED so per-file transfer percentages appear in stdout.
//   - /NFL /NDL suppress the verbose file/dir listing — keeps output focused on % lines.
//   - /E recurse into subdirectories, /R:0 no retries, /W:0 no wait between retries.
func runRobocopy(ctx context.Context, job Job, report ProgressFunc) (int, string) {
	args := []string{job.SourcePath, job.DestPath, "/E", "/R:0", "/W:0", "/NFL", "/NDL"}
	if job.SyncFlags != nil {
		args = append(args, job.SyncFlags.ToRobocopyArgs()...)
	}
	cmd := exec.CommandContext(ctx, "robocopy", args...)
	return streamRobocopy(cmd, report)
}

// runRsync builds and streams an rsync command.
//
// --info=progress2 replaces -v and emits a single overall percentage line,
// making it straightforward to parse without per-file noise.
func runRsync(ctx context.Context, job Job, report ProgressFunc) (int, string) {
	src := strings.TrimRight(job.SourcePath, "/") + "/"
	args := []string{"-a", "--info=progress2", src, job.DestPath}
	if job.SyncFlags != nil {
		args = append(args, job.SyncFlags.ToRsyncArgs()...)
	}
	cmd := exec.CommandContext(ctx, "rsync", args...)
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

	if err := scanner.Err(); err != nil {
		log.Printf("scanner error: %v", err)
		return 1, fmt.Sprintf("output stream error: %v", err)
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

	if err := scanner.Err(); err != nil {
		log.Printf("scanner error: %v", err)
		return 1, fmt.Sprintf("output stream error: %v", err)
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
