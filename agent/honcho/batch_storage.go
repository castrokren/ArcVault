package honcho

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MetricsCollector accumulates job execution metrics throughout the day
// and batches them to Honcho at a scheduled time.
type MetricsCollector struct {
	client      *Client
	executions  []JobExecution
	mu          sync.Mutex
	nextBatchAt time.Time
	batchTime   int // Hour (0-23) to batch each day, e.g., 22 = 10pm
}

// NewMetricsCollector creates a collector that batches at the specified hour.
// batchTime: hour (0-23) to batch, e.g., 22 = 10pm
func NewMetricsCollector(client *Client, batchTime int) *MetricsCollector {
	return &MetricsCollector{
		client:      client,
		executions:  []JobExecution{},
		batchTime:   batchTime,
		nextBatchAt: getNextBatchTime(batchTime),
	}
}

// RecordExecution adds a job execution to the buffer.
func (mc *MetricsCollector) RecordExecution(exec JobExecution) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if exec.Timestamp.IsZero() {
		exec.Timestamp = time.Now()
	}
	mc.executions = append(mc.executions, exec)
}

// ProcessBatch checks if it's time to batch and stores all collected executions.
// Call this periodically (e.g., every minute) to check for batch time.
func (mc *MetricsCollector) ProcessBatch() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Not time yet
	if time.Now().Before(mc.nextBatchAt) {
		return nil
	}

	// Time to batch
	if len(mc.executions) == 0 {
		log.Printf("batch time reached but no executions to store\n")
		mc.nextBatchAt = getNextBatchTime(mc.batchTime)
		return nil
	}

	count := len(mc.executions)
	log.Printf("batch time reached, storing %d execution(s) to honcho\n", count)

	// Store all executions
	if err := mc.client.BatchStoreExecutions(mc.executions); err != nil {
		log.Printf("batch storage failed: %v\n", err)
		// Don't clear on error — retry next batch
		return err
	}

	// Clear after successful store
	mc.executions = []JobExecution{}
	mc.nextBatchAt = getNextBatchTime(mc.batchTime)
	log.Printf("batch stored successfully, next batch at %v\n", mc.nextBatchAt)

	return nil
}

// Flush forces immediate storage of all buffered executions.
func (mc *MetricsCollector) Flush() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if len(mc.executions) == 0 {
		return nil
	}

	log.Printf("flushing %d execution(s) to honcho\n", len(mc.executions))
	if err := mc.client.BatchStoreExecutions(mc.executions); err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}

	mc.executions = []JobExecution{}
	return nil
}

// BufferSize returns number of executions waiting to be stored.
func (mc *MetricsCollector) BufferSize() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return len(mc.executions)
}

// getNextBatchTime calculates the next occurrence of the batch hour.
func getNextBatchTime(batchHour int) time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), batchHour, 0, 0, 0, now.Location())

	// If batch time has already passed today, schedule for tomorrow
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}

	return next
}
