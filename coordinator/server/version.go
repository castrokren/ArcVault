package server

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"
)

// startTime records when the server process started.
var startTime = time.Now()

// BuildTime is optionally injected at build time via ldflags: -X server.BuildTime=...
var BuildTime = "unknown"

// APIContract: matches dashboard/src/types/api.ts VersionResponse interface
// Last synced: 2026-06-03
type versionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	Uptime    string `json:"uptime"`
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	v := os.Getenv("ARCVAULT_VERSION")
	if v == "" {
		v = Version
	}

	uptime := time.Since(startTime).Round(time.Second).String()

	json.NewEncoder(w).Encode(versionResponse{
		Version:   v,
		BuildTime: BuildTime,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		Uptime:    uptime,
	})
}
