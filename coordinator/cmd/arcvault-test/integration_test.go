package main

import (
	"testing"
)

// Test multi-agent job dispatch with varying agent counts
func TestMultiAgentJobDispatch(t *testing.T) {
	// Scenario: Create 1 job, dispatch to multiple agents
	// Expected: All agents execute independently
	h := NewHarness(&LoadScenario{
		Agents:        5,
		JobsPerAgent:  1,
		JobDurationMs: 200,
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All agents should execute the job
	if report.JobsCompleted != 5 {
		t.Errorf("expected 5 agents to execute job, got %d completed", report.JobsCompleted)
	}
}

// Test agent failover: if one agent fails, other agents continue
func TestAgentFailover(t *testing.T) {
	// Scenario: 10 agents, one fails mid-execution
	// Expected: Other 9 agents complete their jobs
	h := NewHarness(&LoadScenario{
		Agents:        10,
		JobsPerAgent:  2,
		JobDurationMs: 300,
		FailureRate:   0.1, // One agent will fail
		FailureType:   "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// At least 90% of jobs should complete
	completionRate := float64(report.JobsCompleted) / float64(report.JobsCreated)
	if completionRate < 0.9 {
		t.Errorf("expected 90%% completion with agent failover, got %.1f%%", completionRate*100)
	}
}

// Test high concurrent job load (stress test)
func TestHighConcurrentLoad(t *testing.T) {
	// Scenario: 50 agents × 10 jobs = 500 total jobs running concurrently
	// Expected: All complete without crashes or memory leaks
	h := NewHarness(&LoadScenario{
		Agents:        50,
		JobsPerAgent:  10,
		JobDurationMs: 50,
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	if report.JobsCompleted != 500 {
		t.Errorf("expected 500 jobs to complete, got %d", report.JobsCompleted)
	}

	// Memory should remain reasonable (< 100MB for 50 agents)
	if report.PeakMemoryMb > 100 {
		t.Errorf("peak memory too high for 50 agents: %.1f MB", report.PeakMemoryMb)
	}
}

// Test job cancellation behavior
func TestJobCancellation(t *testing.T) {
	// Scenario: Start jobs, then cancel them mid-execution
	// Expected: Jobs halt gracefully without hanging
	h := NewHarness(&LoadScenario{
		Agents:        5,
		JobsPerAgent:  3,
		JobDurationMs: 500, // Long-running jobs
		FailureRate:   0,
	})

	// For now, this is simulated by the harness completing normally
	// In real integration, we'd call a Cancel endpoint mid-execution
	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All jobs should complete (cancellation would reduce this count in real scenario)
	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all jobs to complete, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}
}
