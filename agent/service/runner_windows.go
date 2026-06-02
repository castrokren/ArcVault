//go:build windows

package service

import (
	"log"
	"os"

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
func RunService(fn func()) error {
	return svc.Run(AgentServiceName, &agentHandler{run: fn})
}
