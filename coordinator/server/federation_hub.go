package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"arcvault/coordinator/db"

	"github.com/gorilla/websocket"
)

// SubCache holds the last-known state snapshot for a sub-coordinator.
type SubCache struct {
	Agents  []agentResponse `json:"agents"`
	Jobs    []Job           `json:"jobs"`
	History []JobRun        `json:"history"`
	Stale   bool            `json:"stale"`
	AsOf    time.Time       `json:"as_of"`
}

type subConn struct {
	id    string
	conn  *websocket.Conn
	cache *SubCache
	mu    sync.RWMutex
}

// FederationHub manages active sub-coordinator WebSocket connections and their
// in-memory state caches on the root coordinator.
type FederationHub struct {
	db   *db.DB
	subs map[string]*subConn
	mu   sync.RWMutex
}

// NewFederationHub creates a new FederationHub backed by the given database.
func NewFederationHub(database *db.DB) *FederationHub {
	return &FederationHub{
		db:   database,
		subs: make(map[string]*subConn),
	}
}

// HandleSubConnect is the HTTP handler for GET /ws/federation.
func (h *FederationHub) HandleSubConnect(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	if token == "" {
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
		return
	}

	fed, err := h.db.GetFederationByToken(token)
	if err != nil || fed == nil {
		http.Error(w, "invalid federation token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("FederationHub: WS upgrade failed for sub %s: %v", fed.ID, err)
		return
	}

	sc := &subConn{
		id:    fed.ID,
		conn:  conn,
		cache: &SubCache{},
	}

	h.mu.Lock()
	h.subs[fed.ID] = sc
	h.mu.Unlock()

	log.Printf("FederationHub: sub %q (%s) connected", fed.Name, fed.ID)

	_, raw, err := conn.ReadMessage()
	if err != nil {
		log.Printf("FederationHub: failed to read snapshot from sub %s: %v", fed.ID, err)
		h.markOffline(fed.ID)
		conn.Close()
		return
	}

	var msg FedMessage
	if err := json.Unmarshal(raw, &msg); err != nil || msg.Type != FedEventSnapshot {
		log.Printf("FederationHub: expected snapshot from sub %s, got %q", fed.ID, msg.Type)
		h.markOffline(fed.ID)
		conn.Close()
		return
	}

	var snap FedSnapshot
	if err := json.Unmarshal(msg.Payload, &snap); err != nil {
		log.Printf("FederationHub: failed to parse snapshot from sub %s: %v", fed.ID, err)
		h.markOffline(fed.ID)
		conn.Close()
		return
	}

	sc.mu.Lock()
	sc.cache.Agents = snap.Agents
	sc.cache.Jobs = snap.Jobs
	sc.cache.History = snap.History
	sc.cache.Stale = false
	sc.cache.AsOf = time.Now()
	sc.mu.Unlock()

	if err := h.db.SetFederationStatus(fed.ID, "online", time.Now(), snap.Version); err != nil {
		log.Printf("FederationHub: failed to set status online for sub %s: %v", fed.ID, err)
	}

	log.Printf("FederationHub: sub %q snapshot loaded (%d agents, %d jobs)", fed.Name, len(snap.Agents), len(snap.Jobs))

	go func() {
		defer func() {
			h.markOffline(fed.ID)
			conn.Close()
			log.Printf("FederationHub: sub %q disconnected", fed.Name)
		}()

		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var msg FedMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				log.Printf("FederationHub: bad message from sub %s: %v", fed.ID, err)
				continue
			}

			h.applyDelta(sc, msg)
		}
	}()
}

// applyDelta updates the sub's in-memory cache based on an incoming delta event.
func (h *FederationHub) applyDelta(sc *subConn, msg FedMessage) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	switch msg.Type {
	case FedEventAgentHeartbeat:
		var hb FedAgentHeartbeat
		if err := json.Unmarshal(msg.Payload, &hb); err != nil {
			return
		}
		for i, a := range sc.cache.Agents {
			if a.ID == hb.AgentID {
				sc.cache.Agents[i].Status = hb.Status
				sc.cache.Agents[i].LastSeen = hb.LastSeen
				sc.cache.Agents[i].Version = hb.Version
				break
			}
		}

	case FedEventJobStateChange:
		var jsc FedJobStateChange
		if err := json.Unmarshal(msg.Payload, &jsc); err != nil {
			return
		}
		for i, j := range sc.cache.Jobs {
			if j.ID == jsc.JobID {
				sc.cache.Jobs[i].Status = jsc.Status
				break
			}
		}

	case FedEventAgentRegistered:
		var ar FedAgentRegistered
		if err := json.Unmarshal(msg.Payload, &ar); err != nil {
			return
		}
		// Upsert — update in place if agent already exists, append if new.
		found := false
		for i, a := range sc.cache.Agents {
			if a.ID == ar.Agent.ID {
				sc.cache.Agents[i] = ar.Agent
				found = true
				break
			}
		}
		if !found {
			sc.cache.Agents = append(sc.cache.Agents, ar.Agent)
		}

	case FedEventAgentDeleted:
		var ad FedAgentDeleted
		if err := json.Unmarshal(msg.Payload, &ad); err != nil {
			return
		}
		agents := sc.cache.Agents[:0]
		for _, a := range sc.cache.Agents {
			if a.ID != ad.AgentID {
				agents = append(agents, a)
			}
		}
		sc.cache.Agents = agents

	case FedEventSnapshot:
		var snap FedSnapshot
		if err := json.Unmarshal(msg.Payload, &snap); err != nil {
			return
		}
		sc.cache.Agents = snap.Agents
		sc.cache.Jobs = snap.Jobs
		sc.cache.History = snap.History
		sc.cache.Stale = false
		sc.cache.AsOf = time.Now()
	}
}

// markOffline sets the sub's DB status to offline and marks its cache stale.
func (h *FederationHub) markOffline(siteID string) {
	h.mu.Lock()
	sc, ok := h.subs[siteID]
	if ok {
		sc.mu.Lock()
		sc.cache.Stale = true
		sc.cache.AsOf = time.Now()
		sc.mu.Unlock()
		delete(h.subs, siteID)
	}
	h.mu.Unlock()

	if err := h.db.SetFederationStatus(siteID, "offline", time.Now(), ""); err != nil {
		log.Printf("FederationHub: failed to set status offline for sub %s: %v", siteID, err)
	}
}

// GetCache returns a copy of the cache for the given site ID.
func (h *FederationHub) GetCache(siteID string) (*SubCache, bool) {
	h.mu.RLock()
	sc, ok := h.subs[siteID]
	h.mu.RUnlock()
	if !ok {
		return nil, false
	}
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	c := *sc.cache
	return &c, true
}

// AllCaches returns a map of siteID → cache for all currently connected subs.
func (h *FederationHub) AllCaches() map[string]*SubCache {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]*SubCache, len(h.subs))
	for id, sc := range h.subs {
		sc.mu.RLock()
		c := *sc.cache
		sc.mu.RUnlock()
		out[id] = &c
	}
	return out
}

// SendCommand sends a root→sub command to the given site.
func (h *FederationHub) SendCommand(siteID string, cmd FedMessage) error {
	h.mu.RLock()
	sc, ok := h.subs[siteID]
	h.mu.RUnlock()
	if !ok {
		return &subNotConnectedError{siteID: siteID}
	}

	raw, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteMessage(websocket.TextMessage, raw)
}

// DropConnection closes the active WebSocket for the given site (if any).
func (h *FederationHub) DropConnection(siteID string) {
	h.mu.Lock()
	sc, ok := h.subs[siteID]
	if ok {
		sc.conn.Close()
	}
	h.mu.Unlock()
}

type subNotConnectedError struct {
	siteID string
}

func (e *subNotConnectedError) Error() string {
	return "sub-coordinator " + e.siteID + " is not connected"
}
