package main

import (
	"testing"
	"time"
)

// Test that harness can spawn mock agents and create jobs
func TestHarnessBasicLoad(t *testing.T) {
	h := NewHarness(&LoadScenario{
		Agents:         5,
		JobsPerAgent:   3,
		JobDurationMs:  100, // 100ms jobs for fast test
		FailureRate:    0,   // No failures for basic test
	})

	// Run the harness
	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Verify results
	if report.AgentsSpawned != 5 {
		t.Errorf("expected 5 agents spawned, got %d", report.AgentsSpawned)
	}

	if report.JobsCreated != 15 { // 5 agents × 3 jobs
		t.Errorf("expected 15 jobs created, got %d", report.JobsCreated)
	}

	if report.JobsCompleted != 15 {
		t.Errorf("expected 15 jobs completed, got %d", report.JobsCompleted)
	}

	if report.JobsFailed != 0 {
		t.Errorf("expected 0 failed jobs, got %d", report.JobsFailed)
	}

	if report.ThroughputJobsPerSecond <= 0 {
		t.Errorf("throughput should be positive, got %f", report.ThroughputJobsPerSecond)
	}
}

// Test that harness tracks latency
func TestHarnessLatency(t *testing.T) {
	h := NewHarness(&LoadScenario{
		Agents:        3,
		JobsPerAgent:  2,
		JobDurationMs: 50,
		FailureRate:   0,
	})

	report, err := h.Run()
	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	if report.LatencyP50Ms <= 0 {
		t.Errorf("p50 latency should be positive, got %f", report.LatencyP50Ms)
	}

	if report.LatencyP95Ms <= 0 {
		t.Errorf("p95 latency should be positive, got %f", report.LatencyP95Ms)
	}

	if report.LatencyP99Ms <= 0 {
		t.Errorf("p99 latency should be positive, got %f", report.LatencyP99Ms)
	}

	// p99 should be >= p95 >= p50
	if report.LatencyP95Ms < report.LatencyP50Ms {
		t.Errorf("p95 (%f) should be >= p50 (%f)", report.LatencyP95Ms, report.LatencyP50Ms)
	}

	if report.LatencyP99Ms < report.LatencyP95Ms {
		t.Errorf("p99 (%f) should be >= p95 (%f)", report.LatencyP99Ms, report.LatencyP95Ms)
	}
}

// Test that harness measures memory usage
func TestHarnessMemoryTracking(t *testing.T) {
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

	if report.PeakMemoryMb <= 0 {
		t.Errorf("peak memory should be positive, got %f", report.PeakMemoryMb)
	}
}

// Test that harness runs within reasonable time
func TestHarnessCompletionTime(t *testing.T) {
	h := NewHarness(&LoadScenario{
		Agents:        3,
		JobsPerAgent:  2,
		JobDurationMs: 50,
		FailureRate:   0,
	})

	start := time.Now()
	report, err := h.Run()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("harness.Run() failed: %v", err)
	}

	// Should complete in reasonable time (job duration + overhead)
	maxDuration := time.Duration(report.JobDurationMs)*time.Millisecond + 2*time.Second
	if elapsed > maxDuration {
		t.Errorf("harness took %v, expected < %v", elapsed, maxDuration)
	}
}
