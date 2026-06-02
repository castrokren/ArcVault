package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"

	"github.com/gorilla/websocket"
)

// FederationClient runs on a sub-coordinator. When federation config is present
// it dials the root coordinator, sends a full state snapshot on connect, then
// streams delta events as local state changes. Reconnects automatically with
// exponential backoff.
type FederationClient struct {
	cfg    *config.FederationConfig
	db     *db.DB
	ver    string // binary version string, injected at build time

	conn   *websocket.Conn
	connMu sync.Mutex

	stopCh chan struct{}
}

// NewFederationClient creates a client. ver is the coordinator binary version.
func NewFederationClient(cfg *config.FederationConfig, database *db.DB, ver string) *FederationClient {
	return &FederationClient{
		cfg:    cfg,
		db:     database,
		ver:    ver,
		stopCh: make(chan struct{}),
	}
}

// Start begins the connection loop in the current goroutine. Call via go.
func (c *FederationClient) Start() {
	backoff := time.Second
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		if err := c.connect(); err != nil {
			log.Printf("FederationClient: connect failed: %v — retrying in %s", err, backoff)
			select {
			case <-time.After(backoff):
			case <-c.stopCh:
				return
			}
			backoff *= 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			continue
		}

		// Reset backoff on successful connection.
		backoff = time.Second

		// readLoop blocks until disconnect.
		c.readLoop()

		// Clear conn so BroadcastDelta drops messages while reconnecting.
		c.connMu.Lock()
		c.conn = nil
		c.connMu.Unlock()

		log.Printf("FederationClient: disconnected from root — reconnecting")
	}
}

// Stop signals the client to stop reconnecting and exit Start().
func (c *FederationClient) Stop() {
	close(c.stopCh)
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connMu.Unlock()
}

// connect dials the root, authenticates, and sends the opening snapshot.
func (c *FederationClient) connect() error {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.cfg.Token)

	wsURL := wsURL(c.cfg.RootURL) + "/ws/federation"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return err
	}

	if err := c.sendSnapshot(conn); err != nil {
		conn.Close()
		return err
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	log.Printf("FederationClient: connected to root at %s", c.cfg.RootURL)
	return nil
}

// sendSnapshot builds a full state snapshot from the local DB and sends it.
func (c *FederationClient) sendSnapshot(conn *websocket.Conn) error {
	agents, err := c.queryAgents()
	if err != nil {
		return err
	}
	jobs, err := c.queryJobs()
	if err != nil {
		return err
	}
	history, err := c.queryHistory(100)
	if err != nil {
		return err
	}

	snap := FedSnapshot{
		Agents:  agents,
		Jobs:    jobs,
		History: history,
		Version: c.ver,
	}

	payload, err := json.Marshal(snap)
	if err != nil {
		return err
	}

	msg := FedMessage{
		Type:    FedEventSnapshot,
		Payload: json.RawMessage(payload),
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, raw)
}

// readLoop processes root→sub commands until the connection closes.
func (c *FederationClient) readLoop() {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()
	if conn == nil {
		return
	}

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg FedMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("FederationClient: bad message from root: %v", err)
			continue
		}

		c.handleCommand(msg)
	}
}

// handleCommand dispatches root→sub commands to local server logic.
func (c *FederationClient) handleCommand(msg FedMessage) {
	switch msg.Type {
	case FedCmdTriggerJob:
		var p FedCmdTriggerJobPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			log.Printf("FederationClient: bad trigger_job payload: %v", err)
			return
		}
		_, err := c.db.Conn().Exec(
			`UPDATE jobs SET status = 'pending'
			 WHERE id = ? AND status NOT IN ('pending', 'running')`, p.JobID,
		)
		if err != nil {
			log.Printf("FederationClient: trigger_job %s failed: %v", p.JobID, err)
		} else {
			log.Printf("FederationClient: triggered job %s", p.JobID)
		}

	case FedCmdRunTemplate:
		var p FedCmdRunTemplatePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			log.Printf("FederationClient: bad run_template payload: %v", err)
			return
		}
		t, err := c.db.GetTemplate(p.TemplateID)
		if err != nil || t == nil {
			log.Printf("FederationClient: run_template %s not found", p.TemplateID)
			return
		}
		runID := "tpl-fed-" + p.TemplateID + "-" + time.Now().Format("20060102150405")
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = c.db.Conn().Exec(
			`INSERT INTO jobs (id, agent_id, name, source_path, dest_path, command, status, created_at)
			 VALUES (?, ?, ?, '', '', ?, 'pending', ?)`,
			runID, t.AgentID, t.Name, t.Command, now,
		)
		if err != nil {
			log.Printf("FederationClient: run_template insert failed: %v", err)
		} else {
			log.Printf("FederationClient: fired template %s as job %s", p.TemplateID, runID)
		}

	case FedCmdUpdateAgent:
		var p FedCmdUpdateAgentPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			log.Printf("FederationClient: bad update_agent payload: %v", err)
			return
		}
		// Agent update is initiated by the coordinator sending a WS message to the agent.
		// Log for now — full implementation hooks into agent_update.go send path.
		log.Printf("FederationClient: update_agent %s requested by root (not yet wired)", p.AgentID)

	default:
		log.Printf("FederationClient: unknown command type %q", msg.Type)
	}
}

// BroadcastDelta sends a delta event to the root. Silently drops if not connected.
func (c *FederationClient) BroadcastDelta(msg FedMessage) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()
	if conn == nil {
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		log.Printf("FederationClient: broadcast delta failed: %v", err)
	}
}

// --- snapshot query helpers ---

func (c *FederationClient) queryAgents() ([]agentResponse, error) {
	rows, err := c.db.Conn().Query(
		`SELECT id, hostname, os, arch, version, status, last_seen, registered_at, rollback_available
		 FROM agents ORDER BY registered_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []agentResponse
	for rows.Next() {
		var a agentResponse
		var lastSeen *string
		if err := rows.Scan(&a.ID, &a.Hostname, &a.OS, &a.Arch, &a.Version, &a.Status,
			&lastSeen, &a.RegisteredAt, &a.RollbackAvailable); err != nil {
			continue
		}
		a.LastSeen = lastSeen
		agents = append(agents, a)
	}
	if agents == nil {
		agents = []agentResponse{}
	}
	return agents, rows.Err()
}

func (c *FederationClient) queryJobs() ([]Job, error) {
	rows, err := c.db.Conn().Query(
		`SELECT id, agent_id, name, source_path, dest_path, schedule, status, created_at
		 FROM jobs ORDER BY created_at DESC LIMIT 500`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		var schedule *string
		if err := rows.Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath,
			&schedule, &j.Status, &j.CreatedAt); err != nil {
			continue
		}
		j.Schedule = schedule
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []Job{}
	}
	return jobs, rows.Err()
}

func (c *FederationClient) queryHistory(limit int) ([]JobRun, error) {
	rows, err := c.db.Conn().Query(
		`SELECT id, job_id, exit_code, output, finished_at
		 FROM job_runs ORDER BY finished_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []JobRun
	for rows.Next() {
		var r JobRun
		var output, finishedAt *string
		if err := rows.Scan(&r.ID, &r.JobID, &r.ExitCode, &output, &finishedAt); err != nil {
			continue
		}
		if output != nil {
			r.Output = *output
		}
		if finishedAt != nil {
			r.FinishedAt = *finishedAt
		}
		runs = append(runs, r)
	}
	if runs == nil {
		runs = []JobRun{}
	}
	return runs, rows.Err()
}

// wsURL converts an http/https base URL to ws/wss.
func wsURL(base string) string {
	if len(base) >= 8 && base[:8] == "https://" {
		return "wss://" + base[8:]
	}
	if len(base) >= 7 && base[:7] == "http://" {
		return "ws://" + base[7:]
	}
	return base
}
