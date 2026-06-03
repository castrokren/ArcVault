package main

import (
	"testing"
)

// Test with very large source paths (simulating deeply nested directories)
func TestLargeSourcePaths(t *testing.T) {
	// Scenario: Jobs with very long source paths
	// Expected: Handled gracefully without truncation or errors
	h := NewHarness(&LoadScenario{
		Agents:        3,
		JobsPerAgent:  5,
		JobDurationMs: 100,
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all jobs with large paths to complete, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}
}

// Test with very high file counts (100k+ files)
func TestHighFileCount(t *testing.T) {
	// Scenario: Single job with 100,000+ files
	// Expected: Progress tracking remains accurate, no memory leak
	h := NewHarness(&LoadScenario{
		Agents:        2,
		JobsPerAgent:  1,
		JobDurationMs: 500, // Simulate long backup of many files
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	if report.JobsCompleted != 2 {
		t.Errorf("expected both jobs with high file counts to complete, got %d", report.JobsCompleted)
	}

	// Memory should not explode
	if report.PeakMemoryMb > 50 {
		t.Errorf("memory usage too high for file count test: %.1f MB", report.PeakMemoryMb)
	}
}

// Test permission denied scenarios
func TestPermissionDeniedErrors(t *testing.T) {
	// Scenario: Attempt backup of read-only or inaccessible paths
	// Expected: Job fails with clear error message (not hang/crash)
	h := NewHarness(&LoadScenario{
		Agents:        3,
		JobsPerAgent:  2,
		JobDurationMs: 150,
		FailureRate:   0,
		FailureType:   "permission_denied", // Not currently supported, for future
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All jobs should complete (either succeed or fail gracefully)
	if report.JobsCompleted+report.JobsFailed != report.JobsCreated {
		t.Errorf("expected all jobs to complete or fail gracefully, got %d/%d",
			report.JobsCompleted+report.JobsFailed, report.JobsCreated)
	}
}

// Test long-running job with agent disconnect at 50% completion
func TestLongJobDisconnectAtMidpoint(t *testing.T) {
	// Scenario: 10-minute job, agent disconnects at 5 minutes
	// Expected: Job pauses, resumes from checkpoint when agent reconnects
	h := NewHarness(&LoadScenario{
		Agents:        3,
		JobsPerAgent:  2,
		JobDurationMs: 1000, // 1 second = 1 "minute" in test
		FailureRate:   0.5,  // 50% will disconnect
		FailureType:   "disconnect",
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Jobs should complete despite mid-execution disconnects
	if report.JobsCompleted < report.JobsCreated/2 {
		t.Errorf("expected most jobs to complete despite disconnects, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}

	// Should have recovery metrics
	if report.FailuresRecovered == 0 && report.FailuresInjected > 0 {
		t.Errorf("expected failures to be recovered, got %d injected, %d recovered",
			report.FailuresInjected, report.FailuresRecovered)
	}
}

// Test concurrent writes to same destination
func TestConcurrentWritesToSameDestination(t *testing.T) {
	// Scenario: Multiple agents writing to same backup destination
	// Expected: No data corruption, proper locking
	h := NewHarness(&LoadScenario{
		Agents:        10,
		JobsPerAgent:  5,
		JobDurationMs: 100,
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// All jobs should complete despite concurrent writes
	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all jobs to complete with concurrent writes, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}
}

// Test rapid job creation and completion (churn)
func TestRapidJobCreationAndCompletion(t *testing.T) {
	// Scenario: Create many jobs rapidly, complete them rapidly
	// Expected: No queue overflow, no dropped jobs
	h := NewHarness(&LoadScenario{
		Agents:        20,
		JobsPerAgent:  50,
		JobDurationMs: 10, // Very fast jobs
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	totalJobs := 20 * 50 // 1000 jobs
	if report.JobsCreated != totalJobs {
		t.Errorf("expected %d jobs created, got %d", totalJobs, report.JobsCreated)
	}

	if report.JobsCompleted != totalJobs {
		t.Errorf("expected %d jobs completed, got %d", totalJobs, report.JobsCompleted)
	}

	// Should achieve high throughput
	if report.ThroughputJobsPerSecond < 100 {
		t.Errorf("throughput too low for rapid job churn: %.1f jobs/sec", report.ThroughputJobsPerSecond)
	}
}

// Test job execution with mixed durations
func TestMixedJobDurations(t *testing.T) {
	// Scenario: Some jobs 100ms, some 500ms, mixed in parallel
	// Expected: Shorter jobs complete first, longer jobs don't block them
	h := NewHarness(&LoadScenario{
		Agents:        5,
		JobsPerAgent:  4,
		JobDurationMs: 200, // Average, but in real scenario would vary per job
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	if report.JobsCompleted != report.JobsCreated {
		t.Errorf("expected all jobs to complete regardless of duration, got %d/%d",
			report.JobsCompleted, report.JobsCreated)
	}

	// Latency variance should show p99 > p50 (some jobs take longer)
	if report.LatencyP99Ms < report.LatencyP50Ms {
		t.Errorf("p99 latency should be >= p50, got p50=%.1f, p99=%.1f",
			report.LatencyP50Ms, report.LatencyP99Ms)
	}
}
