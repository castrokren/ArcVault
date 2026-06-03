package main

import (
	"testing"
)

// Test job recovery when coordinator restarts
func TestJobRecoveryOnCoordinatorRestart(t *testing.T) {
	// Scenario: Job in-flight when coordinator crashes and restarts
	// Expected: Job resumed from progress checkpoint in DB
	h := NewHarness(&LoadScenario{
		Agents:        5,
		JobsPerAgent:  3,
		JobDurationMs: 500,
		FailureRate:   0.15, // 15% will trigger coordinator crash
		FailureType:   "crash",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Most jobs should recover after restart
	completionRate := float64(report.JobsCompleted) / float64(report.JobsCreated)
	if completionRate < 0.85 {
		t.Errorf("expected 85%% recovery rate after coordinator restart, got %.1f%%", completionRate*100)
	}
}

// Test multiple agents reconnecting simultaneously
func TestMultipleAgentReconnection(t *testing.T) {
	// Scenario: 5 agents disconnect at the same time (network partition)
	// Expected: Each reconnects independently, jobs resume
	h := NewHarness(&LoadScenario{
		Agents:        5,
		JobsPerAgent:  4,
		JobDurationMs: 300,
		FailureRate:   1.0, // All agents will disconnect
		FailureType:   "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All jobs should eventually complete despite simultaneous disconnects
	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all jobs to complete after simultaneous disconnects, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}

	// High recovery rate expected
	if report.FailuresRecovered < report.FailuresInjected*9/10 {
		t.Errorf("expected 90%% recovery rate for disconnects, got %d/%d",
			report.FailuresRecovered, report.FailuresInjected)
	}
}

// Test staggered agent reconnection (cascading recovery)
func TestStaggeredAgentReconnection(t *testing.T) {
	// Scenario: Agents disconnect at different times, reconnect at different times
	// Expected: Jobs resume as each agent reconnects
	h := NewHarness(&LoadScenario{
		Agents:        10,
		JobsPerAgent:  2,
		JobDurationMs: 250,
		FailureRate:   0.3,
		FailureType:   "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Most jobs should complete
	if report.JobsCompleted < report.JobsCreated*8/10 {
		t.Errorf("expected 80%% completion with staggered disconnects, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}
}

// Test recovery from database transaction failures
func TestDatabaseTransactionRecovery(t *testing.T) {
	// Scenario: High concurrent writes cause database transaction failures
	// Expected: Transactions retry and eventually succeed
	h := NewHarness(&LoadScenario{
		Agents:        15,
		JobsPerAgent:  5,
		JobDurationMs: 75,
		FailureRate:   0, // No injected failures, just natural contention
		FailureType:   "none",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All jobs should complete despite database contention
	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all jobs to complete despite transaction contention, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}
}

// Test job progress persistence across agent disconnect/reconnect
func TestProgressPersistenceAcrossDisconnect(t *testing.T) {
	// Scenario: Agent executing job, progress at 50%, agent disconnects
	// Expected: Job pauses, progress saved to DB, resumes from 50% when agent reconnects
	h := NewHarness(&LoadScenario{
		Agents:        3,
		JobsPerAgent:  3,
		JobDurationMs: 600, // Long enough for progress to be meaningful
		FailureRate:   0.25,
		FailureType:   "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Jobs should complete with reasonable latency
	// (not restarted from beginning, but resumed from checkpoint)
	if report.JobsCompleted < report.JobsCreated*7/10 {
		t.Errorf("expected progress to be preserved, got %d/%d jobs completed",
			report.JobsCompleted, report.JobsCreated)
	}

	// Average latency should be close to job duration (not 2x)
	if report.LatencyP95Ms > float64(2*h.scenario.JobDurationMs) {
		t.Errorf("latency suggests jobs restarted rather than resumed: p95=%.1fms (duration=%dms)",
			report.LatencyP95Ms, h.scenario.JobDurationMs)
	}
}

// Test coordinator failover in federation scenario
func TestCoordinatorFailoverInFederation(t *testing.T) {
	// Scenario: Primary coordinator fails, secondary takes over
	// Expected: Jobs continue on secondary coordinator
	h := NewHarness(&LoadScenario{
		Agents:        8,
		JobsPerAgent:  3,
		JobDurationMs: 400,
		FailureRate:   0.1, // Simulate coordinator failover
		FailureType:   "crash",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Jobs should continue running on secondary
	if report.JobsCompleted < report.JobsCreated*9/10 {
		t.Errorf("expected 90%% of jobs to survive coordinator failover, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}
}
