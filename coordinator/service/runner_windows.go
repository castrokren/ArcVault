//go:build windows

package service

import (
	"io/fs"
	"log"
	"os"

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

	// Launch the HTTP server in a goroutine so we return to SCM immediately.
	// http.ListenAndServe blocks; we surface its exit via a channel.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- cmd.StartCommand(h.cfg, h.staticFS)
	}()

	// Tell SCM the service is up and what control signals we accept.
	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for {
		select {
		case err := <-serverErr:
			// The HTTP server exited on its own (unexpected).
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
				log.Println("ArcVault coordinator: stop signal received, shutting down")
				status <- svc.Status{State: svc.StopPending}
				// http.ListenAndServe has no cancel path without refactoring server.go.
				// os.Exit is acceptable here; the OS cleans up file handles and SQLite.
				os.Exit(0)
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
