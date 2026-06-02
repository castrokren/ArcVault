package runner

import (
	"strings"
	"testing"
)

// Test: SyncFlags struct exists with robocopy and rsync options
func TestSyncFlagsStructure(t *testing.T) {
	flags := &SyncFlags{
		Mirror:       true,
		MaxAge:       30,       // days
		MinAge:       1,        // days
		MaxSize:      1024,     // MB
		ExcludeFiles: []string{"*.tmp", "*.log"},
		ExcludeDirs:  []string{".git", "node_modules"},
	}

	if !flags.Mirror {
		t.Error("expected Mirror flag to be true")
	}
	if flags.MaxAge != 30 {
		t.Errorf("expected MaxAge 30, got %d", flags.MaxAge)
	}
	if len(flags.ExcludeFiles) != 2 {
		t.Errorf("expected 2 exclude files, got %d", len(flags.ExcludeFiles))
	}
}

// Test: Validate rejects invalid MaxAge (negative)
func TestValidateSyncFlagsMaxAgeNegative(t *testing.T) {
	flags := &SyncFlags{
		MaxAge: -1,
	}
	err := flags.Validate()
	if err == nil {
		t.Error("expected error for negative MaxAge")
	}
	if !strings.Contains(err.Error(), "MaxAge") {
		t.Errorf("expected error to mention MaxAge, got: %v", err)
	}
}

// Test: Validate rejects invalid MinAge (negative)
func TestValidateSyncFlagsMinAgeNegative(t *testing.T) {
	flags := &SyncFlags{
		MinAge: -5,
	}
	err := flags.Validate()
	if err == nil {
		t.Error("expected error for negative MinAge")
	}
}

// Test: Validate rejects MaxSize (negative)
func TestValidateSyncFlagsMaxSizeNegative(t *testing.T) {
	flags := &SyncFlags{
		MaxSize: -100,
	}
	err := flags.Validate()
	if err == nil {
		t.Error("expected error for negative MaxSize")
	}
}

// Test: Validate rejects MinAge > MaxAge
func TestValidateSyncFlagsMinGreaterThanMax(t *testing.T) {
	flags := &SyncFlags{
		MinAge: 30,
		MaxAge: 10,
	}
	err := flags.Validate()
	if err == nil {
		t.Error("expected error when MinAge > MaxAge")
	}
	if !strings.Contains(err.Error(), "MinAge") || !strings.Contains(err.Error(), "MaxAge") {
		t.Errorf("expected error to mention both MinAge and MaxAge, got: %v", err)
	}
}

// Test: Validate accepts valid flags
func TestValidateSyncFlagsValid(t *testing.T) {
	flags := &SyncFlags{
		Mirror:       true,
		MaxAge:       30,
		MinAge:       1,
		MaxSize:      1024,
		ExcludeFiles: []string{"*.tmp"},
		ExcludeDirs:  []string{".git"},
	}
	err := flags.Validate()
	if err != nil {
		t.Errorf("expected no error for valid flags, got: %v", err)
	}
}

// Test: ToRobocopyArgs builds correct robocopy arguments
func TestToRobocopyArgsBasic(t *testing.T) {
	flags := &SyncFlags{
		Mirror: true,
	}
	args := flags.ToRobocopyArgs()

	found := false
	for _, arg := range args {
		if arg == "/MIR" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected /MIR in robocopy args, got: %v", args)
	}
}

// Test: ToRobocopyArgs builds MAXAGE flag correctly
func TestToRobocopyArgsMaxAge(t *testing.T) {
	flags := &SyncFlags{
		MaxAge: 30,
	}
	args := flags.ToRobocopyArgs()

	foundMaxAge := false
	for _, arg := range args {
		if arg == "/MAXAGE:30" {
			foundMaxAge = true
			break
		}
	}
	if !foundMaxAge {
		t.Errorf("expected /MAXAGE:30 in robocopy args, got: %v", args)
	}
}

// Test: ToRobocopyArgs builds MINAGE flag correctly
func TestToRobocopyArgsMinAge(t *testing.T) {
	flags := &SyncFlags{
		MinAge: 5,
	}
	args := flags.ToRobocopyArgs()

	foundMinAge := false
	for _, arg := range args {
		if arg == "/MINAGE:5" {
			foundMinAge = true
			break
		}
	}
	if !foundMinAge {
		t.Errorf("expected /MINAGE:5 in robocopy args, got: %v", args)
	}
}

// Test: ToRobocopyArgs builds MAXSIZE flag correctly
func TestToRobocopyArgsMaxSize(t *testing.T) {
	flags := &SyncFlags{
		MaxSize: 1024,
	}
	args := flags.ToRobocopyArgs()

	foundMaxSize := false
	for _, arg := range args {
		if arg == "/MAXSIZE:1024M" {
			foundMaxSize = true
			break
		}
	}
	if !foundMaxSize {
		t.Errorf("expected /MAXSIZE:1024M in robocopy args, got: %v", args)
	}
}

// Test: ToRobocopyArgs builds exclude file patterns correctly
func TestToRobocopyArgsExcludeFiles(t *testing.T) {
	flags := &SyncFlags{
		ExcludeFiles: []string{"*.tmp", "*.log"},
	}
	args := flags.ToRobocopyArgs()

	foundTmp := false
	foundLog := false
	for _, arg := range args {
		if arg == "/XF" {
			// Next args should be the patterns
			continue
		}
		if arg == "*.tmp" {
			foundTmp = true
		}
		if arg == "*.log" {
			foundLog = true
		}
	}
	if !foundTmp || !foundLog {
		t.Errorf("expected *.tmp and *.log in robocopy args, got: %v", args)
	}
}

// Test: ToRobocopyArgs builds exclude directory patterns correctly
func TestToRobocopyArgsExcludeDirs(t *testing.T) {
	flags := &SyncFlags{
		ExcludeDirs: []string{".git", "node_modules"},
	}
	args := flags.ToRobocopyArgs()

	foundGit := false
	foundNode := false
	for _, arg := range args {
		if arg == ".git" {
			foundGit = true
		}
		if arg == "node_modules" {
			foundNode = true
		}
	}
	if !foundGit || !foundNode {
		t.Errorf("expected .git and node_modules in robocopy args, got: %v", args)
	}
}

// Test: ToRsyncArgs builds correct rsync arguments
func TestToRsyncArgsBasic(t *testing.T) {
	flags := &SyncFlags{
		Mirror: true,
	}
	args := flags.ToRsyncArgs()

	found := false
	for _, arg := range args {
		if arg == "--delete" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --delete in rsync args for Mirror=true, got: %v", args)
	}
}

// Test: ToRsyncArgs builds --max-age flag correctly
func TestToRsyncArgsMaxAge(t *testing.T) {
	flags := &SyncFlags{
		MaxAge: 30,
	}
	args := flags.ToRsyncArgs()

	found := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "--max-age=") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --max-age in rsync args, got: %v", args)
	}
}

// Test: ToRsyncArgs builds --min-age flag correctly
func TestToRsyncArgsMinAge(t *testing.T) {
	flags := &SyncFlags{
		MinAge: 5,
	}
	args := flags.ToRsyncArgs()

	found := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "--min-age=") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --min-age in rsync args, got: %v", args)
	}
}

// Test: ToRsyncArgs builds --maxsize flag correctly
func TestToRsyncArgsMaxSize(t *testing.T) {
	flags := &SyncFlags{
		MaxSize: 1024,
	}
	args := flags.ToRsyncArgs()

	found := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "--maxsize=") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --maxsize in rsync args, got: %v", args)
	}
}

// Test: ToRsyncArgs builds exclude patterns correctly
func TestToRsyncArgsExcludePatterns(t *testing.T) {
	flags := &SyncFlags{
		ExcludeFiles: []string{"*.tmp", "*.log"},
		ExcludeDirs:  []string{".git"},
	}
	args := flags.ToRsyncArgs()

	foundTmp := false
	foundLog := false
	foundGit := false

	for _, arg := range args {
		if strings.Contains(arg, "*.tmp") {
			foundTmp = true
		}
		if strings.Contains(arg, "*.log") {
			foundLog = true
		}
		if strings.Contains(arg, ".git") {
			foundGit = true
		}
	}
	if !foundTmp || !foundLog || !foundGit {
		t.Errorf("expected exclude patterns in rsync args, got: %v", args)
	}
}

// Test: Empty SyncFlags produces no extra arguments
func TestSyncFlagsEmptyProducesNoArgs(t *testing.T) {
	flags := &SyncFlags{}

	robocopyArgs := flags.ToRobocopyArgs()
	if len(robocopyArgs) > 0 {
		t.Errorf("expected no robocopy args for empty SyncFlags, got: %v", robocopyArgs)
	}

	rsyncArgs := flags.ToRsyncArgs()
	if len(rsyncArgs) > 0 {
		t.Errorf("expected no rsync args for empty SyncFlags, got: %v", rsyncArgs)
	}
}

// Test: Job struct can hold SyncFlags
func TestJobWithSyncFlags(t *testing.T) {
	job := &Job{
		ID:         "job-123",
		AgentID:    "agent-1",
		Name:       "backup",
		SourcePath: "C:\\data",
		DestPath:   "D:\\backup",
		SyncFlags: &SyncFlags{
			Mirror: true,
			MaxAge: 30,
		},
	}

	if job.SyncFlags == nil {
		t.Error("expected SyncFlags to be set on Job")
	}
	if !job.SyncFlags.Mirror {
		t.Error("expected Mirror to be true")
	}
}
