package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialTestWS connects a gorilla websocket client to the test server's /ws endpoint.
func dialTestWS(t *testing.T, s *Server) (*websocket.Conn, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(s.router)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{
		"Authorization": []string{authHeader()},
	})
	if err != nil {
		ts.Close()
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	return conn, ts
}

// readEvent reads one JSON event from the WebSocket connection with a timeout.
func readEvent(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read WebSocket message: %v", err)
	}
	var event map[string]interface{}
	if err := json.Unmarshal(msg, &event); err != nil {
		t.Fatalf("WebSocket message is not valid JSON: %v\nraw: %s", err, msg)
	}
	return event
}

// --- hub unit tests ---

func TestHub_broadcastDeliveredToConnectedClient(t *testing.T) {
	s := newTestServer(t)
	conn, ts := dialTestWS(t, s)
	defer ts.Close()
	defer conn.Close()

	// give hub time to register the client
	time.Sleep(50 * time.Millisecond)

	s.hub.Broadcast(Event{Type: "test.ping", Payload: map[string]string{"hello": "world"}})

	event := readEvent(t, conn)
	if event["type"] != "test.ping" {
		t.Errorf("expected type 'test.ping', got %q", event["type"])
	}
}

func TestHub_broadcastDeliveredToMultipleClients(t *testing.T) {
	s := newTestServer(t)

	conn1, ts := dialTestWS(t, s)
	defer ts.Close()
	defer conn1.Close()

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn2, _, err := websocket.DefaultDialer.Dial(url, http.Header{
		"Authorization": []string{authHeader()},
	})
	if err != nil {
		t.Fatalf("failed to connect second client: %v", err)
	}
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	s.hub.Broadcast(Event{Type: "test.multi", Payload: map[string]string{"x": "y"}})

	e1 := readEvent(t, conn1)
	e2 := readEvent(t, conn2)

	if e1["type"] != "test.multi" {
		t.Errorf("client1: expected 'test.multi', got %q", e1["type"])
	}
	if e2["type"] != "test.multi" {
		t.Errorf("client2: expected 'test.multi', got %q", e2["type"])
	}
}

func TestHub_noErrorWhenNoClientsConnected(t *testing.T) {
	s := newTestServer(t)
	// should not panic or block with zero clients
	s.hub.Broadcast(Event{Type: "test.empty", Payload: nil})
}

// --- integration: job status change triggers broadcast ---

func TestWebSocket_jobClaimedEventBroadcastOnStatusUpdate(t *testing.T) {
	s := newTestServer(t)
	conn, ts := dialTestWS(t, s)
	defer ts.Close()
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// create a job
	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", strings.NewReader(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	// patch status to running — should trigger broadcast
	patch := `{"status":"running"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+created.ID+"/status", strings.NewReader(patch))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	event := readEvent(t, conn)
	if event["type"] != "job.updated" {
		t.Errorf("expected type 'job.updated', got %q", event["type"])
	}
	payload, ok := event["payload"].(map[string]interface{})
	if !ok {
		t.Fatal("expected payload to be an object")
	}
	if payload["id"] != created.ID {
		t.Errorf("expected payload.id %q, got %q", created.ID, payload["id"])
	}
	if payload["status"] != "running" {
		t.Errorf("expected payload.status 'running', got %q", payload["status"])
	}
}

func TestWebSocket_jobResultsEventBroadcastOnResultPost(t *testing.T) {
	s := newTestServer(t)
	conn, ts := dialTestWS(t, s)
	defer ts.Close()
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// create a job
	body := `{"agent_id":"agent-01","name":"backup","source_path":"C:\\src","dest_path":"D:\\backup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", strings.NewReader(body))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	var created Job
	json.NewDecoder(rr.Body).Decode(&created)

	// post result — should trigger broadcast
	result := `{"exit_code":0,"output":"done"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.ID+"/results", strings.NewReader(result))
	req2.Header.Set("Authorization", authHeader())
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req2)

	event := readEvent(t, conn)
	if event["type"] != "job.result" {
		t.Errorf("expected type 'job.result', got %q", event["type"])
	}
	payload, ok := event["payload"].(map[string]interface{})
	if !ok {
		t.Fatal("expected payload to be an object")
	}
	if payload["job_id"] != created.ID {
		t.Errorf("expected payload.job_id %q, got %q", created.ID, payload["job_id"])
	}
}
