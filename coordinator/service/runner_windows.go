//go:build windows

package service

import (
	"io/fs"
	"log"
	"os"
	"time"

	"arcvault/coordinator/cmd"
	"arcvault/coordinator/config"
	"golang.org/x/sys/windows/svc"
)

// coordinatorHandler implements svc.Handler so the coordinator can run
// under the Windows Service Control Manager.
type coordinatorHandler struct {
	cfg      *config.Config
	staticFS fs.FS
}

// Execute is called by svc.Run. It must signal StartPending, launch work,
// signal Running, then loop on the control channel until Stop/Shutdown.
func (h *coordinatorHandler) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	// Create stop channel for graceful shutdown
	stopCh := make(chan struct{})

	// Launch the HTTP server in a goroutine so we return to SCM immediately.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- cmd.StartCommandWithContext(h.cfg, h.staticFS, stopCh)
	}()

	// Tell SCM the service is up and what control signals we accept.
	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for {
		select {
		case err := <-serverErr:
			// The HTTP server exited (expected on shutdown).
			if err != nil {
				log.Printf("coordinator server error: %v", err)
			}
			status <- svc.Status{State: svc.StopPending}
			return false, 0

		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				// SCM is asking for our current status — echo it back.
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Println("ArcVault coordinator: stop signal received, shutting down gracefully")
				status <- svc.Status{State: svc.StopPending}
				// Signal server to stop and wait with timeout
				close(stopCh)
				select {
				case err := <-serverErr:
					if err != nil {
						log.Printf("coordinator shutdown error: %v", err)
					}
					return false, 0
				case <-time.After(10 * time.Second):
					// Timeout — force exit
					log.Println("shutdown timeout, forcing exit")
					return false, 0
				}
			default:
				log.Printf("ArcVault coordinator: unexpected SCM control request %d", c.Cmd)
			}
		}
	}
}

// RunService hands control to the Windows Service Control Manager.
// Call this from the "run-service" subcommand — not from "start".
func RunService(cfg *config.Config, staticFS fs.FS) error {
	os.Setenv("ARCVAULT_SERVICE", "1")
	return svc.Run(CoordinatorServiceName, &coordinatorHandler{cfg: cfg, staticFS: staticFS})
}
