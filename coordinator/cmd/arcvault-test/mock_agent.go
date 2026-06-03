package main

import (
	"fmt"
	"time"
)

// MockAgent simulates a real backup agent
type MockAgent struct {
	harness   *Harness
	id        int
	name      string
	connected bool
}

// NewMockAgent creates a new mock agent
func NewMockAgent(harness *Harness, id int) *MockAgent {
	return &MockAgent{
		harness:   harness,
		id:        id,
		name:      fmt.Sprintf("test-agent-%d", id),
		connected: true,
	}
}

// Run executes the agent, creating and running the specified number of jobs
func (a *MockAgent) Run(numJobs int, jobDurationMs int) {
	// Simulate agent registration (connect)
	a.Register()
	defer a.Disconnect()

	// Execute jobs
	for i := 0; i < numJobs; i++ {
		if !a.connected {
			// In a real scenario, the agent would be reconnecting here
			// For now, just skip if disconnected
			continue
		}

		a.ExecuteJob(i, jobDurationMs)
	}
}

// Register registers the agent with the harness
func (a *MockAgent) Register() {
	a.connected = true
	a.harness.IncConnection()
}

// Disconnect unregisters the agent from the harness
func (a *MockAgent) Disconnect() {
	a.connected = false
	a.harness.DecConnection()
}

// ExecuteJob simulates executing a single job
func (a *MockAgent) ExecuteJob(jobIndex int, durationMs int) {
	startTime := time.Now()

	// Check if a failure should be injected
	shouldFail := a.harness.injector.ShouldInjectFailure()
	recovered := false

	if shouldFail {
		// Inject failure (disconnect, timeout, crash, etc.)
		a.harness.injector.InjectFailure(a.id, jobIndex)

		// In a real scenario, this would pause and then resume
		// Simulate: pause for a bit, then resume execution
		halfDuration := durationMs / 2
		time.Sleep(time.Duration(halfDuration) * time.Millisecond)

		// Recover (reconnect, retry, etc.)
		recovered = true

		// Resume job execution
		remainingDuration := durationMs - halfDuration
		time.Sleep(time.Duration(remainingDuration) * time.Millisecond)
	} else {
		// Normal job execution
		time.Sleep(time.Duration(durationMs) * time.Millisecond)
	}

	endTime := time.Now()

	// Record the metric
	a.harness.RecordJobMetric(
		JobMetric{
			StartTime: startTime,
			EndTime:   endTime,
			Failed:    shouldFail,
			Recovered: recovered,
		},
	)
}

// ID returns the agent's ID
func (a *MockAgent) ID() string {
	return a.name
}

// IsConnected returns whether the agent is currently connected
func (a *MockAgent) IsConnected() bool {
	return a.connected
}
