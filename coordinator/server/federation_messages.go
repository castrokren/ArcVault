package server

import "encoding/json"

// FedMessage is the wire envelope for all federation WebSocket messages.
type FedMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Sub → Root event types.
const (
	FedEventSnapshot        = "snapshot"
	FedEventAgentHeartbeat  = "agent_heartbeat"
	FedEventJobStateChange  = "job_state_change"
	FedEventAgentRegistered = "agent_registered"
	FedEventAgentDeleted    = "agent_deleted"
	FedEventTemplateChanged = "template_changed"
)

// Root → Sub command types.
const (
	FedCmdTriggerJob  = "trigger_job"
	FedCmdRunTemplate = "run_template"
	FedCmdUpdateAgent = "update_agent"
)

// FedSnapshot is the full-state payload sent by the sub on connect/reconnect.
type FedSnapshot struct {
	Agents  []agentResponse `json:"agents"`
	Jobs    []Job           `json:"jobs"`
	History []JobRun        `json:"history"`
	Version string          `json:"version"`
}

// FedAgentHeartbeat is the delta payload for agent status updates.
type FedAgentHeartbeat struct {
	AgentID  string  `json:"agent_id"`
	Status   string  `json:"status"`
	LastSeen *string `json:"last_seen"`
	Version  string  `json:"version"`
}

// FedJobStateChange is the delta payload for job state transitions.
type FedJobStateChange struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// FedAgentRegistered is the delta payload when a new agent joins the sub.
type FedAgentRegistered struct {
	Agent agentResponse `json:"agent"`
}

// FedAgentDeleted is the delta payload when an agent is removed from the sub.
type FedAgentDeleted struct {
	AgentID string `json:"agent_id"`
}

// FedTemplateChanged is the delta payload for template create/update/delete.
type FedTemplateChanged struct {
	Action     string `json:"action"` // "created" | "updated" | "deleted"
	TemplateID string `json:"template_id"`
}

// FedCmdTriggerJobPayload is the command payload for triggering a job on the sub.
type FedCmdTriggerJobPayload struct {
	JobID string `json:"job_id"`
}

// FedCmdRunTemplatePayload is the command payload for running a template on the sub.
type FedCmdRunTemplatePayload struct {
	TemplateID string `json:"template_id"`
}

// FedCmdUpdateAgentPayload is the command payload for updating an agent on the sub.
type FedCmdUpdateAgentPayload struct {
	AgentID string `json:"agent_id"`
}
