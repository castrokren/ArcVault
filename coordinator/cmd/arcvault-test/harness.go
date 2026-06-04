package main

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"
)

// LoadScenario defines the parameters for a load test
type LoadScenario struct {
	Agents        int     // Number of mock agents to spawn
	JobsPerAgent  int     // Number of jobs each agent should execute
	JobDurationMs int     // Duration of each job in milliseconds
	FailureRate   float64 // Probability of failure (0.0 to 1.0)
	FailureType   string  // "disconnect", "timeout", "crash", etc.
}

// TestReport contains the results of a load test
type TestReport struct {
	AgentsSpawned            int       `json:"agents_spawned"`
	JobsCreated              int       `json:"jobs_created"`
	JobsCompleted            int       `json:"jobs_completed"`
	JobsFailed               int       `json:"jobs_failed"`
	DurationSeconds          float64   `json:"duration_seconds"`
	ThroughputJobsPerSecond  float64   `json:"throughput_jobs_per_second"`
	LatencyP50Ms             float64   `json:"latency_p50_ms"`
	LatencyP95Ms             float64   `json:"latency_p95_ms"`
	LatencyP99Ms             float64   `json:"latency_p99_ms"`
	PeakMemoryMb             float64   `json:"peak_memory_mb"`
	JobDurationMs            int       `json:"job_duration_ms"`
	FailuresInjected         int       `json:"failures_injected"`
	FailuresRecovered        int       `json:"failures_recovered"`
	AvgConnectionCount       float64   `json:"avg_connection_count"`
	MaxConnectionCount       int       `json:"max_connection_count"`
}

// JobMetric tracks timing for a single job
type JobMetric struct {
	StartTime time.Time
	EndTime   time.Time
	Failed    bool
	Recovered bool
}

// Harness orchestrates a load test
type Harness struct {
	scenario    *LoadScenario
	agents      []*MockAgent
	metrics     []JobMetric
	metricsMu   sync.Mutex
	startTime   time.Time
	connCount   int
	connCountMu sync.Mutex
	injector    *FailureInjector
}

// NewHarness creates a new harness with the given scenario
func NewHarness(scenario *LoadScenario) *Harness {
	return &Harness{
		scenario:  scenario,
		agents:    make([]*MockAgent, 0, scenario.Agents),
		metrics:   make([]JobMetric, 0, scenario.Agents*scenario.JobsPerAgent),
		injector:  NewFailureInjector(scenario),
	}
}

// Run executes the load test and returns a report
func (h *Harness) Run() (*TestReport, error) {
	h.startTime = time.Now()

	// Spawn mock agents
	var wg sync.WaitGroup
	for i := 0; i < h.scenario.Agents; i++ {
		agent := NewMockAgent(h, i)
		h.agents = append(h.agents, agent)
		wg.Add(1)

		go func(a *MockAgent) {
			defer wg.Done()
			a.Run(h.scenario.JobsPerAgent, h.scenario.JobDurationMs)
		}(agent)
	}

	// Wait for all agents to complete
	wg.Wait()

	// Generate report
	return h.generateReport(), nil
}

// RecordJobMetric records the timing of a job execution
func (h *Harness) RecordJobMetric(metric JobMetric) {
	h.metricsMu.Lock()
	defer h.metricsMu.Unlock()
	h.metrics = append(h.metrics, metric)
}

// IncConnection increments the active connection count
func (h *Harness) IncConnection() {
	h.connCountMu.Lock()
	defer h.connCountMu.Unlock()
	h.connCount++
}

// DecConnection decrements the active connection count
func (h *Harness) DecConnection() {
	h.connCountMu.Lock()
	defer h.connCountMu.Unlock()
	h.connCount--
}

// GetConnectionCount returns the current connection count
func (h *Harness) GetConnectionCount() int {
	h.connCountMu.Lock()
	defer h.connCountMu.Unlock()
	return h.connCount
}

// generateReport creates a TestReport from collected metrics
func (h *Harness) generateReport() *TestReport {
	h.metricsMu.Lock()
	defer h.metricsMu.Unlock()

	elapsed := time.Since(h.startTime)

	// Calculate job statistics
	var latencies []float64
	completedCount := 0
	failedCount := 0
	recoveredCount := 0

	for _, m := range h.metrics {
		// A job is completed if it either succeeded outright or failed but recovered
		if !m.Failed || m.Recovered {
			completedCount++
		} else {
			failedCount++
		}

		// Track recoveries separately
		if m.Failed && m.Recovered {
			recoveredCount++
		}

		duration := m.EndTime.Sub(m.StartTime).Seconds() * 1000
		latencies = append(latencies, duration)
	}

	// Calculate percentiles
	sort.Float64s(latencies)
	p50, p95, p99 := float64(0), float64(0), float64(0)
	if len(latencies) > 0 {
		p50 = latencies[len(latencies)*50/100]
		p95 = latencies[len(latencies)*95/100]
		p99 = latencies[len(latencies)*99/100]
	}

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	peakMemoryMb := float64(m.Alloc) / 1024 / 1024

	// Throughput
	throughput := float64(len(h.metrics)) / elapsed.Seconds()

	// Get failure stats from injector
	injected, recovered, _ := h.injector.RecoveryStats()

	return &TestReport{
		AgentsSpawned:           h.scenario.Agents,
		JobsCreated:             len(h.metrics),
		JobsCompleted:           completedCount,
		JobsFailed:              failedCount,
		DurationSeconds:         elapsed.Seconds(),
		ThroughputJobsPerSecond: throughput,
		LatencyP50Ms:            p50,
		LatencyP95Ms:            p95,
		LatencyP99Ms:            p99,
		PeakMemoryMb:            peakMemoryMb,
		JobDurationMs:           h.scenario.JobDurationMs,
		FailuresInjected:        injected,
		FailuresRecovered:       recovered,
	}
}

// String returns a human-readable summary of the report
func (r *TestReport) String() string {
	return fmt.Sprintf(`
Load Test Report
================
Agents:              %d
Jobs Created:        %d
Jobs Completed:      %d
Jobs Failed:         %d
Duration:            %.2fs
Throughput:          %.2f jobs/sec
Latency (p50/p95/p99): %.1f / %.1f / %.1fms
Peak Memory:         %.1f MB
`,
		r.AgentsSpawned,
		r.JobsCreated,
		r.JobsCompleted,
		r.JobsFailed,
		r.DurationSeconds,
		r.ThroughputJobsPerSecond,
		r.LatencyP50Ms,
		r.LatencyP95Ms,
		r.LatencyP99Ms,
		r.PeakMemoryMb,
	)
}
