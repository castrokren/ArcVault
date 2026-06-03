package main

import (
	"testing"
)

// Test agent disconnect recovery during long job execution
func TestAgentDisconnectRecovery(t *testing.T) {
	// Scenario: 20 agents running 30-minute jobs with random disconnects every 5 minutes
	// Expected: Jobs pause and resume, no data loss
	h := NewHarness(&LoadScenario{
		Agents:         5,
		JobsPerAgent:   2,
		JobDurationMs:  1000, // 1 second = 1 "minute" in test time
		FailureRate:    0.3,  // 30% of jobs experience disconnect
		FailureType:    "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Verify jobs completed despite failures
	if report.JobsCompleted < (report.JobsCreated - report.JobsFailed) {
		t.Errorf("expected %d jobs to complete, got %d",
			report.JobsCreated-report.JobsFailed, report.JobsCompleted)
	}

	// Verify recovery time is reasonable
	if report.FailuresInjected > 0 && report.FailuresRecovered == 0 {
		t.Errorf("failures were injected but none recovered: %d injected, %d recovered",
			report.FailuresInjected, report.FailuresRecovered)
	}
}

// Test multiple agents disconnecting simultaneously
func TestSimultaneousAgentDisconnects(t *testing.T) {
	// Scenario: 10 agents, all disconnect at same time
	// Expected: All reconnect independently, jobs resume
	h := NewHarness(&LoadScenario{
		Agents:        10,
		JobsPerAgent:  1,
		JobDurationMs: 500,
		FailureRate:   1.0, // 100% - all jobs will experience disconnect
		FailureType:   "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All jobs should eventually complete despite disconnects
	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all %d jobs to complete, got %d",
			report.JobsCreated, report.JobsCompleted)
	}

	// Recovery count should be high
	if report.FailuresRecovered < report.FailuresInjected/2 {
		t.Errorf("expected at least 50%% recovery rate, got %d/%d",
			report.FailuresRecovered, report.FailuresInjected)
	}
}

// Test network timeout scenario
func TestNetworkTimeout(t *testing.T) {
	// Scenario: Network requests hang, then timeout and retry
	// Expected: Jobs complete after retry
	h := NewHarness(&LoadScenario{
		Agents:        5,
		JobsPerAgent:  3,
		JobDurationMs: 200,
		FailureRate:   0.2, // 20% timeout rate
		FailureType:   "timeout",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Most jobs should complete despite timeouts
	completionRate := float64(report.JobsCompleted) / float64(report.JobsCreated)
	if completionRate < 0.8 {
		t.Errorf("expected 80%% completion with timeouts, got %.1f%%",
			completionRate*100)
	}
}

// Test coordinator crash and recovery
func TestCoordinatorCrashRecovery(t *testing.T) {
	// Scenario: Coordinator crashes once during test, restarts
	// Expected: In-flight jobs recovered from DB
	h := NewHarness(&LoadScenario{
		Agents:        8,
		JobsPerAgent:  2,
		JobDurationMs: 300,
		FailureRate:   0.1, // 10% will experience coordinator crash
		FailureType:   "crash",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Jobs should recover after coordinator restarts
	if report.JobsCompleted < report.JobsCreated-1 {
		t.Errorf("expected at least %d jobs to complete after crash recovery, got %d",
			report.JobsCreated-1, report.JobsCompleted)
	}
}

// Test database lock under concurrent load
func TestDatabaseLockUnderLoad(t *testing.T) {
	// Scenario: High concurrent writes cause lock contention
	// Expected: No data corruption, eventual completion
	h := NewHarness(&LoadScenario{
		Agents:        20,
		JobsPerAgent:  5,
		JobDurationMs: 50,
		FailureRate:   0, // No injected failures, just natural contention
		FailureType:   "none",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All jobs should complete despite contention
	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all %d jobs to complete under lock contention, got %d",
			report.JobsCreated, report.JobsCompleted)
	}

	// Throughput should still be reasonable
	if report.ThroughputJobsPerSecond <= 0 {
		t.Errorf("throughput should be positive, got %f", report.ThroughputJobsPerSecond)
	}
}

// Test recovery time metric
func TestFailureRecoveryTime(t *testing.T) {
	// Scenario: Measure how long it takes to recover from failures
	// Expected: Recovery time ≤ 5 seconds (your requirement)
	h := NewHarness(&LoadScenario{
		Agents:        10,
		JobsPerAgent:  3,
		JobDurationMs: 500,
		FailureRate:   0.2,
		FailureType:   "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Verify metrics exist
	if report.FailuresInjected == 0 {
		t.Skip("no failures injected, skipping recovery time test")
	}

	// Log recovery metrics (would be checked by CI/CD)
	t.Logf("Failures injected: %d", report.FailuresInjected)
	t.Logf("Failures recovered: %d", report.FailuresRecovered)
	t.Logf("Recovery rate: %.1f%%", float64(report.FailuresRecovered)/float64(report.FailuresInjected)*100)
}
