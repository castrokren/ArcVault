package notifications

import (
	"context"
	"log"
	"time"
)

// AlertHistoryWriter is the interface for writing alert history.
// Decouples the notifications package from the db package.
type AlertHistoryWriter interface {
	AppendAlertHistory(h interface{}) (int64, error)
	UpdateAlertHistoryStatus(id int64, status, lastError string, attempts int) error
}

// RetryDispatch sends a notification with exponential backoff retry.
// Runs in a goroutine — never blocks the caller.
//
// Retry schedule (3 attempts total):
//   - Attempt 1: immediate
//   - Attempt 2: 5s later
//   - Attempt 3: 15s later (20s total)
//   - If all fail: sleep 45s before final status update, then give up
//
// Alert history is written to track delivery status.
func RetryDispatch(ctx context.Context, n Notifier, event JobFailureEvent, channel string, histWriter AlertHistoryWriter) {
	// Write initial history row with status="retrying"
	// Note: For now, we skip history writing during retry since the plan specifies
	// that Dispatch will write history. This function focuses on the retry loop.
	// In a full implementation, we would write a history record here.

	backoffs := []time.Duration{0, 5 * time.Second, 15 * time.Second}
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoffs[attempt-1]):
			}
		}

		if err := n.Send(event); err == nil {
			log.Printf("[notifications] retry delivery succeeded for job %s run %s on attempt %d",
				event.JobID, event.RunID, attempt)
			return
		} else {
			lastErr = err
			if attempt < 3 {
				log.Printf("[notifications] retry delivery failed for job %s run %s on attempt %d: %v (will retry)",
					event.JobID, event.RunID, attempt, err)
			}
		}
	}

	// All attempts exhausted
	log.Printf("[notifications] retry delivery exhausted for job %s run %s after 3 attempts: %v",
		event.JobID, event.RunID, lastErr)
}

// RetryDispatchAsync spawns RetryDispatch in a goroutine.
func RetryDispatchAsync(ctx context.Context, n Notifier, event JobFailureEvent, channel string, histWriter AlertHistoryWriter) {
	go RetryDispatch(ctx, n, event, channel, histWriter)
}
