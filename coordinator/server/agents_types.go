package server

import (
	"fmt"
	"regexp"
)

// isValidUUID checks if a string is a valid UUID v4
func isValidUUID(s string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(s)
}

// isValidOS checks if OS is a supported value
func isValidOS(os string) bool {
	return os == "linux" || os == "darwin" || os == "windows"
}

// isValidArch checks if architecture is a valid value
func isValidArch(arch string) bool {
	validArchs := []string{"amd64", "arm64", "386", "arm", "ppc64le", "s390x"}
	for _, v := range validArchs {
		if arch == v {
			return true
		}
	}
	return false
}

// isValidSemver checks if version matches semver format
func isValidSemver(version string) bool {
	semverRegex := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-[a-zA-Z0-9.-]+)?$`)
	return semverRegex.MatchString(version)
}

// RegisterRequest defines the request to register/update an agent
type RegisterRequest struct {
	AgentID  string `json:"agent_id"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
}

// Validate checks if RegisterRequest is valid
func (r *RegisterRequest) Validate() error {
	if r.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if !isValidUUID(r.AgentID) {
		return fmt.Errorf("agent_id must be a valid UUID v4")
	}
	if r.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if len(r.Hostname) < 1 || len(r.Hostname) > 255 {
		return fmt.Errorf("hostname must be 1-255 characters")
	}
	if r.OS == "" {
		return fmt.Errorf("os is required")
	}
	if !isValidOS(r.OS) {
		return fmt.Errorf("os must be 'linux', 'darwin', or 'windows'")
	}
	if r.Arch == "" {
		return fmt.Errorf("arch is required")
	}
	if !isValidArch(r.Arch) {
		return fmt.Errorf("arch must be a valid architecture (amd64, arm64, 386, arm, ppc64le, s390x)")
	}
	if r.Version == "" {
		return fmt.Errorf("version is required")
	}
	if !isValidSemver(r.Version) {
		return fmt.Errorf("version must be in semver format (e.g., v0.1.0 or 1.2.3)")
	}
	return nil
}

// RegisterResponse defines the response after agent registration
type RegisterResponse struct {
	AgentID      string `json:"agent_id"`
	Status       string `json:"status"`
	LastSeen     string `json:"last_seen"`
	RegisteredAt string `json:"registered_at"`
}

// HeartbeatRequest defines the agent heartbeat request
type HeartbeatRequest struct {
	RollbackAvailable bool `json:"rollback_available"`
}

// HeartbeatResponse defines the response to a heartbeat
type HeartbeatResponse struct {
	Status   string `json:"status"`
	LastSeen string `json:"last_seen"`
}

// AgentResponse defines the complete agent information
type AgentResponse struct {
	AgentID           string  `json:"agent_id"`
	Hostname          string  `json:"hostname"`
	OS                string  `json:"os"`
	Arch              string  `json:"arch"`
	Version           string  `json:"version"`
	Status            string  `json:"status"`
	LastSeen          *string `json:"last_seen"`
	RegisteredAt      string  `json:"registered_at"`
	RollbackAvailable bool    `json:"rollback_available"`
}

// PaginatedAgentsResponse wraps paginated agents list
type PaginatedAgentsResponse struct {
	Data       []AgentResponse `json:"data"`
	Pagination PaginationMeta  `json:"pagination"`
}
