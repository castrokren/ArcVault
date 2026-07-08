package runner

import (
	"testing"
)

func TestIsWhitelisted_AcceptsRsync(t *testing.T) {
	if !IsWhitelisted("rsync") {
		t.Error("expected rsync to be whitelisted")
	}
}

func TestIsWhitelisted_AcceptsRobocopy(t *testing.T) {
	if !IsWhitelisted("robocopy") {
		t.Error("expected robocopy to be whitelisted")
	}
}

func TestIsWhitelisted_AcceptsCaseInsensitive(t *testing.T) {
	tests := []string{"RSYNC", "Rsync", "ROBOCOPY", "Robocopy"}
	for _, prog := range tests {
		if !IsWhitelisted(prog) {
			t.Errorf("expected %s to be whitelisted (case-insensitive)", prog)
		}
	}
}

func TestIsWhitelisted_RejectsBash(t *testing.T) {
	if IsWhitelisted("bash") {
		t.Error("expected bash to be rejected")
	}
}

func TestIsWhitelisted_RejectsSh(t *testing.T) {
	if IsWhitelisted("sh") {
		t.Error("expected sh to be rejected")
	}
}

func TestIsWhitelisted_RejectsPowershell(t *testing.T) {
	if IsWhitelisted("powershell") {
		t.Error("expected powershell to be rejected")
	}
}

func TestExtractProgramName_SimpleProgram(t *testing.T) {
	name := ExtractProgramName("rsync -a src dest")
	if name != "rsync" {
		t.Errorf("expected rsync, got %s", name)
	}
}

func TestExtractProgramName_FullPath(t *testing.T) {
	name := ExtractProgramName("/usr/bin/rsync -a src dest")
	if name != "rsync" {
		t.Errorf("expected rsync, got %s", name)
	}
}

func TestExtractProgramName_WindowsPath(t *testing.T) {
	// Windows path with quoted argument (proper way)
	name := ExtractProgramName(`"C:\Program Files\robocopy" src dest`)
	if name != "robocopy" {
		t.Errorf("expected robocopy, got %s", name)
	}
}

func TestExtractProgramName_WindowsPathUnquoted(t *testing.T) {
	// Windows path without spaces (unquoted)
	name := ExtractProgramName(`C:\Windows\robocopy src dest`)
	if name != "robocopy" {
		t.Errorf("expected robocopy, got %s", name)
	}
}

func TestExtractProgramName_WindowsPathBackslash(t *testing.T) {
	name := ExtractProgramName(`C:\Windows\System32\robocopy.exe /E src dest`)
	if name != "robocopy.exe" {
		t.Errorf("expected robocopy.exe, got %s", name)
	}
}

func TestExtractProgramName_WithQuotedArgs(t *testing.T) {
	name := ExtractProgramName(`rsync -a "source dir" "dest dir"`)
	if name != "rsync" {
		t.Errorf("expected rsync, got %s", name)
	}
}

func TestExtractProgramName_WithSingleQuotedArgs(t *testing.T) {
	name := ExtractProgramName(`bash -c 'echo hello'`)
	if name != "bash" {
		t.Errorf("expected bash, got %s", name)
	}
}

func TestExtractProgramName_Empty(t *testing.T) {
	name := ExtractProgramName("")
	if name != "" {
		t.Errorf("expected empty string, got %s", name)
	}
}

func TestExtractProgramName_OnlyWhitespace(t *testing.T) {
	name := ExtractProgramName("   ")
	if name != "" {
		t.Errorf("expected empty string, got %s", name)
	}
}

func TestExtractProgramName_MultipleArgs(t *testing.T) {
	name := ExtractProgramName("rsync -aP --delete src dest /exclude=pattern")
	if name != "rsync" {
		t.Errorf("expected rsync, got %s", name)
	}
}

func TestExtractProgramName_WindowsExe(t *testing.T) {
	name := ExtractProgramName("robocopy.exe /E src dest")
	if name != "robocopy.exe" {
		t.Errorf("expected robocopy.exe, got %s", name)
	}
}
