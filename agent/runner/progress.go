package runner

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// ProgressFunc is called by the executor as progress is parsed from command output.
// pct is the estimated completion percentage (0–100).
// logs are any meaningful output lines collected since the last call.
type ProgressFunc func(pct int, logs []string)

// Noop is a ProgressFunc that discards all progress.
var Noop ProgressFunc = func(int, []string) {}

// ── Pure parsers (no I/O — easily unit-tested) ────────────────────────────────

// robocopyPctRe matches robocopy's per-file progress tokens such as "  23%" or " 100%".
// Robocopy emits these as \r-separated values within a transfer; after splitting on
// \r/\n each token is a bare percentage string.
var robocopyPctRe = regexp.MustCompile(`^\s*(\d{1,3})%\s*$`)

// ParseRobocopyLine parses one token from robocopy stdout after splitting on \r/\n.
//
//   - pct is the per-file transfer percentage (0–100).
//   - isFileComplete is true when pct==100, indicating that one file finished.
//   - ok is false for lines that are not percentage tokens (file names, headers, etc.).
func ParseRobocopyLine(s string) (pct int, isFileComplete bool, ok bool) {
	m := robocopyPctRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, false, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 0 || n > 100 {
		return 0, false, false
	}
	return n, n == 100, true
}

// rsyncProgress2Re matches rsync --info=progress2 lines.
// Example line: "      3,221,688  33%    3.87MB/s    0:00:00"
// The percentage here is the overall job completion, not per-file.
var rsyncProgress2Re = regexp.MustCompile(`\s+(\d{1,3})%\s+[\d.]+\w+/s`)

// ParseRsyncLine parses one line from rsync --info=progress2 stdout.
// Returns (pct, true) when the line carries an overall percentage, or (0, false) otherwise.
func ParseRsyncLine(s string) (pct int, ok bool) {
	m := rsyncProgress2Re.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 0 || n > 100 {
		return 0, false
	}
	return n, true
}

// SplitOnCRLF is a bufio.SplitFunc that splits on bare \r, bare \n, or \r\n pairs.
// Robocopy uses bare \r to overwrite in-place percentage counters on one line,
// so standard line scanning misses those tokens. This splitter handles all cases.
func SplitOnCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\r' || b == '\n' {
			advance = i + 1
			// Treat \r\n as a single separator.
			if b == '\r' && advance < len(data) && data[advance] == '\n' {
				advance++
			}
			return advance, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// Ensure SplitOnCRLF satisfies bufio.SplitFunc at compile time.
var _ bufio.SplitFunc = SplitOnCRLF
