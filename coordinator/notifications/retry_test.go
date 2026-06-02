package notifications

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockNotifier fails N times, then succeeds.
type MockNotifier struct {
	FailCount int
	CallCount int
}

func (m *MockNotifier) Send(event JobFailureEvent) error {
	m.CallCount++
	if m.CallCount <= m.FailCount {
		return errors.New("mock failure")
	}
	return nil
}

func TestRetryDispatch_SucceedsOnFirstAttempt(t *testing.T) {
	m := &MockNotifier{FailCount: 0}
	ctx := context.Background()
	event := JobFailureEvent{
		JobID:   "job1",
		RunID:   "run1",
		AgentID: "agent1",
	}

	RetryDispatch(ctx, m, event, "webhook", nil)

	if m.CallCount != 1 {
		t.Fatalf("expected 1 call, got %d", m.CallCount)
	}
}

func TestRetryDispatch_SucceedsOnSecondAttempt(t *testing.T) {
	m := &MockNotifier{FailCount: 1}
	ctx := context.Background()
	event := JobFailureEvent{
		JobID:   "job1",
		RunID:   "run1",
		AgentID: "agent1",
	}

	before := time.Now()
	RetryDispatch(ctx, m, event, "webhook", nil)
	elapsed := time.Since(before)

	if m.CallCount != 2 {
		t.Fatalf("expected 2 calls, got %d", m.CallCount)
	}
	// Should have waited ~5 seconds
	if elapsed < 4*time.Second {
		t.Fatalf("expected ~5s delay, got %v", elapsed)
	}
}

func TestRetryDispatch_SucceedsOnThirdAttempt(t *testing.T) {
	m := &MockNotifier{FailCount: 2}
	ctx := context.Background()
	event := JobFailureEvent{
		JobID:   "job1",
		RunID:   "run1",
		AgentID: "agent1",
	}

	before := time.Now()
	RetryDispatch(ctx, m, event, "webhook", nil)
	elapsed := time.Since(before)

	if m.CallCount != 3 {
		t.Fatalf("expected 3 calls, got %d", m.CallCount)
	}
	// Should have waited ~5s + ~15s = ~20s
	if elapsed < 19*time.Second {
		t.Fatalf("expected ~20s delay, got %v", elapsed)
	}
}

func TestRetryDispatch_ExhaustedAfterThreeAttempts(t *testing.T) {
	m := &MockNotifier{FailCount: 99} // Always fails
	ctx := context.Background()
	event := JobFailureEvent{
		JobID:   "job1",
		RunID:   "run1",
		AgentID: "agent1",
	}

	RetryDispatch(ctx, m, event, "webhook", nil)

	if m.CallCount != 3 {
		t.Fatalf("expected 3 calls (exhausted), got %d", m.CallCount)
	}
}

func TestRetryDispatch_ContextCancellation(t *testing.T) {
	m := &MockNotifier{FailCount: 99}
	ctx, cancel := context.WithCancel(context.Background())
	event := JobFailureEvent{
		JobID:   "job1",
		RunID:   "run1",
		AgentID: "agent1",
	}

	// Cancel after first attempt
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	RetryDispatch(ctx, m, event, "webhook", nil)

	if m.CallCount > 2 {
		t.Fatalf("expected context to stop retries early, but got %d calls", m.CallCount)
	}
}
