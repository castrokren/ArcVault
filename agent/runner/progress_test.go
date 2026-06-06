package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ── ParseRobocopyLine ─────────────────────────────────────────────────────────

func TestParseRobocopyLine_ValidPercentages(t *testing.T) {
	tests := []struct {
		input   string
		wantPct int
	}{
		{"  23%", 23},
		{" 100%", 100},
		{"0%", 0},
		{"  99%", 99},
		{"\t56%\t", 56},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pct, _, ok := ParseRobocopyLine(tt.input)
			if !ok {
				t.Fatalf("ParseRobocopyLine(%q) ok=false, want true", tt.input)
			}
			if pct != tt.wantPct {
				t.Errorf("pct=%d want=%d", pct, tt.wantPct)
			}
		})
	}
}

func TestParseRobocopyLine_FileComplete(t *testing.T) {
	pct, done, ok := ParseRobocopyLine(" 100%")
	if !ok || pct != 100 || !done {
		t.Errorf("100%% line: pct=%d done=%v ok=%v; want pct=100 done=true ok=true", pct, done, ok)
	}

	pct, done, ok = ParseRobocopyLine("  50%")
	if !ok || pct != 50 || done {
		t.Errorf("50%% line: pct=%d done=%v ok=%v; want pct=50 done=false ok=true", pct, done, ok)
	}
}

func TestParseRobocopyLine_NonPercentLines(t *testing.T) {
	nonMatch := []string{
		"",
		"New File          1.2 m\tbackup\\file.txt",
		"\tNew File  \t\t 10.0 m\tsubdir\\file.txt",
		"  23% extra text after",
		"robocopy output header",
		"-------------------------------------------------------------------------------",
		"   ROBOCOPY     ::     Robust File Copy for Windows",
		"Files :         5         5         0",
	}
	for _, s := range nonMatch {
		t.Run(fmt.Sprintf("%q", s), func(t *testing.T) {
			_, _, ok := ParseRobocopyLine(s)
			if ok {
				t.Errorf("ParseRobocopyLine(%q) ok=true, want false", s)
			}
		})
	}
}

// ── ParseRsyncLine ────────────────────────────────────────────────────────────

func TestParseRsyncLine_ValidProgress(t *testing.T) {
	tests := []struct {
		input   string
		wantPct int
	}{
		{"      3,221,688  33%    3.87MB/s    0:00:00", 33},
		{"      9,699,328 100%    3.81MB/s    0:00:02 (xfr#1, to-chk=2/4)", 100},
		{"          1,024   0%    0.00kB/s    0:00:00", 0},
		{"    512,000,000  75%   10.24MB/s    0:00:05", 75},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("pct=%d", tt.wantPct), func(t *testing.T) {
			pct, ok := ParseRsyncLine(tt.input)
			if !ok {
				t.Fatalf("ParseRsyncLine(%q) ok=false, want true", tt.input)
			}
			if pct != tt.wantPct {
				t.Errorf("pct=%d want=%d", pct, tt.wantPct)
			}
		})
	}
}

func TestParseRsyncLine_NoMatch(t *testing.T) {
	noMatch := []string{
		"",
		"sent 1,234 bytes  received 567 bytes  1,234.00 bytes/sec",
		"total size is 9,699,328  speedup is 1.00",
		"rsync error: some error (code 23)",
		"building file list ...",
		"./",
		"path/to/file.txt",
	}
	for _, s := range noMatch {
		t.Run(fmt.Sprintf("%q", s), func(t *testing.T) {
			_, ok := ParseRsyncLine(s)
			if ok {
				t.Errorf("ParseRsyncLine(%q) ok=true, want false", s)
			}
		})
	}
}

// ── SplitOnCRLF ──────────────────────────────────────────────────────────────

func TestSplitOnCRLF_BareCarriageReturn(t *testing.T) {
	// Robocopy emits: "  23%\r  56%\r 100%\n"
	input := "  23%\r  56%\r 100%\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitOnCRLF)

	var tokens []string
	for scanner.Scan() {
		tokens = append(tokens, scanner.Text())
	}

	expected := []string{"  23%", "  56%", " 100%"}
	if len(tokens) != len(expected) {
		t.Fatalf("got tokens %v (len=%d), want %v", tokens, len(tokens), expected)
	}
	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token[%d]=%q want=%q", i, tok, expected[i])
		}
	}
}

func TestSplitOnCRLF_CRLFPair(t *testing.T) {
	// Windows-style \r\n should count as one separator.
	input := "line1\r\nline2\r\nline3"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitOnCRLF)

	var tokens []string
	for scanner.Scan() {
		tokens = append(tokens, scanner.Text())
	}

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %v", len(tokens), tokens)
	}
	for i, want := range []string{"line1", "line2", "line3"} {
		if tokens[i] != want {
			t.Errorf("token[%d]=%q want=%q", i, tokens[i], want)
		}
	}
}

func TestSplitOnCRLF_PlainNewlines(t *testing.T) {
	input := "a\nb\nc"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitOnCRLF)

	var tokens []string
	for scanner.Scan() {
		tokens = append(tokens, scanner.Text())
	}

	if len(tokens) != 3 || tokens[0] != "a" || tokens[1] != "b" || tokens[2] != "c" {
		t.Errorf("unexpected tokens: %v", tokens)
	}
}

// ── progressReporter ─────────────────────────────────────────────────────────

// TestProgressReporter_Throttle verifies rapid calls produce at most one HTTP
// request per second.
func TestProgressReporter_Throttle(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pr := newProgressReporter("job-throttle", srv.URL, "token", srv.Client())

	// Ten rapid calls at 50% — only the first should fire (throttled).
	for i := 0; i < 10; i++ {
		pr.Report(50, []string{fmt.Sprintf("log %d", i)})
	}

	if calls != 1 {
		t.Errorf("expected 1 HTTP call (throttled), got %d", calls)
	}
	// Remaining 9 log lines should be buffered in pending.
	if len(pr.pending) != 9 {
		t.Errorf("expected 9 buffered log lines, got %d", len(pr.pending))
	}
}

// TestProgressReporter_AlwaysSendsAt100 verifies pct==100 bypasses the throttle.
func TestProgressReporter_AlwaysSendsAt100(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pr := newProgressReporter("job-100", srv.URL, "token", srv.Client())
	pr.lastSent = time.Now() // simulate a very recent send to engage the throttle

	pr.Report(100, []string{"done"})

	if calls != 1 {
		t.Errorf("pct==100 should always send regardless of throttle, got %d calls", calls)
	}
}

// TestProgressReporter_BuffersLogs verifies log lines accumulate across throttled
// calls and are all flushed together on the next unthrottled send.
func TestProgressReporter_BuffersLogs(t *testing.T) {
	var allLogs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Logs []string `json:"logs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			allLogs = append(allLogs, body.Logs...)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pr := newProgressReporter("job-logs", srv.URL, "token", srv.Client())

	pr.Report(50, []string{"log a"})           // first call — sends immediately
	pr.Report(60, []string{"log b", "log c"}) // throttled — buffered
	pr.Report(100, []string{"log d"})          // forced flush — sends buffered + new

	// Expect: send1=["log a"], send2=["log b","log c","log d"]
	if len(allLogs) != 4 {
		t.Errorf("expected 4 total log lines across two sends, got %d: %v", len(allLogs), allLogs)
	}
}

// TestProgressReporter_StatusIsCompletedAt100 verifies the coordinator receives
// status="completed" when pct==100.
func TestProgressReporter_StatusIsCompletedAt100(t *testing.T) {
	var gotStatus string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Status string `json:"status"`
		}
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		gotStatus = body.Status
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pr := newProgressReporter("job-status", srv.URL, "token", srv.Client())
	pr.Report(100, []string{})

	if gotStatus != "completed" {
		t.Errorf("expected status=%q at pct==100, got %q", "completed", gotStatus)
	}
}

// TestProgressReporter_StatusIsRunningBelow100 verifies status="running" for
// all percentages below 100.
func TestProgressReporter_StatusIsRunningBelow100(t *testing.T) {
	var gotStatus string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Status string `json:"status"`
		}
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		gotStatus = body.Status
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pr := newProgressReporter("job-running", srv.URL, "token", srv.Client())
	pr.Report(42, []string{})

	if gotStatus != "running" {
		t.Errorf("expected status=%q at pct=42, got %q", "running", gotStatus)
	}
}
