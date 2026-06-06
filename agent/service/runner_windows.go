//go:build windows

package service

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
)

type agentHandler struct {
	run func()
}

// Execute is called by svc.Run. It signals StartPending, launches the agent
// goroutine, signals Running, then loops on SCM control requests.
func (h *agentHandler) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	// Run the agent in a goroutine so we return to SCM immediately.
	go h.run()

	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			status <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			log.Println("ArcVault agent: stop signal received, shutting down")
			status <- svc.Status{State: svc.StopPending}
			os.Exit(0)
		default:
			log.Printf("ArcVault agent: unexpected SCM control request %d", c.Cmd)
		}
	}
}

// RunService hands control to the Windows Service Control Manager,
// running fn (the agent loop) inside the SCM-managed process.
//
// A log file is opened at <exe-dir>/arcvault-agent.log so that log output
// is visible even when SCM discards stdout/stderr. Errors opening the log
// file are non-fatal — logging falls back to the default (stderr).
func RunService(fn func()) error {
	setupLogFile()
	return svc.Run(AgentServiceName, &agentHandler{run: fn})
}

// setupLogFile redirects the default logger to <exe-dir>/logs/arcvault-agent.log.
// The logs/ directory is created if it does not exist. Errors are non-fatal;
// logging falls back to stderr.
func setupLogFile() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	logsDir := filepath.Join(filepath.Dir(exe), "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return
	}
	logPath := filepath.Join(logsDir, "arcvault-agent.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("=== ArcVault Agent log opened %s ===", time.Now().Format(time.RFC3339))
}
