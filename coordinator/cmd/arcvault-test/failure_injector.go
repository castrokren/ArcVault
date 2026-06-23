package main

import (
	"math/rand"
	"sync"
	"time"
)

// FailureInjector manages injection of failures into mock agents
type FailureInjector struct {
	scenario        *LoadScenario
	failureEvents   []FailureEvent
	failureEventsMu sync.RWMutex
}

// FailureEvent records a failure that was injected
type FailureEvent struct {
	AgentID       int
	JobIndex      int
	FailureType   string
	TimeInjected  time.Time
	TimeRecovered time.Time
	Recovered     bool
}

// NewFailureInjector creates a new failure injector
func NewFailureInjector(scenario *LoadScenario) *FailureInjector {
	return &FailureInjector{
		scenario:      scenario,
		failureEvents: make([]FailureEvent, 0),
	}
}

// ShouldInjectFailure determines if a failure should be injected for this job
func (fi *FailureInjector) ShouldInjectFailure() bool {
	if fi.scenario.FailureRate <= 0 {
		return false
	}
	return rand.Float64() < fi.scenario.FailureRate
}

// InjectFailure simulates a failure at a specific point in job execution
func (fi *FailureInjector) InjectFailure(agentID int, jobIndex int) FailureType {
	failureType := fi.scenario.FailureType

	switch failureType {
	case "disconnect":
		return fi.injectDisconnect(agentID, jobIndex)
	case "timeout":
		return fi.injectTimeout(agentID, jobIndex)
	case "crash":
		return fi.injectCoordinatorCrash(agentID, jobIndex)
	default:
		return FailureTypeNone
	}
}

// injectDisconnect simulates agent disconnection
func (fi *FailureInjector) injectDisconnect(agentID int, jobIndex int) FailureType {
	event := FailureEvent{
		AgentID:      agentID,
		JobIndex:     jobIndex,
		FailureType:  "disconnect",
		TimeInjected: time.Now(),
	}

	fi.failureEventsMu.Lock()
	defer fi.failureEventsMu.Unlock()
	fi.failureEvents = append(fi.failureEvents, event)

	// Simulate: close websocket, job pauses
	// In a real scenario, the agent would detect this and pause job execution
	// Then it would reconnect and resume

	// For testing: simulate reconnection after a short delay
	go func() {
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

		fi.failureEventsMu.Lock()
		// Find and mark the event as recovered
		for i, e := range fi.failureEvents {
			if e.AgentID == agentID && e.JobIndex == jobIndex {
				fi.failureEvents[i].TimeRecovered = time.Now()
				fi.failureEvents[i].Recovered = true
			}
		}
		fi.failureEventsMu.Unlock()
	}()

	return FailureTypeDisconnect
}

// injectTimeout simulates network timeout
func (fi *FailureInjector) injectTimeout(agentID int, jobIndex int) FailureType {
	event := FailureEvent{
		AgentID:      agentID,
		JobIndex:     jobIndex,
		FailureType:  "timeout",
		TimeInjected: time.Now(),
	}

	fi.failureEventsMu.Lock()
	defer fi.failureEventsMu.Unlock()
	fi.failureEvents = append(fi.failureEvents, event)

	// Simulate: request hangs for 30s, then times out and retries
	go func() {
		time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

		fi.failureEventsMu.Lock()
		// Mark as recovered (retry succeeded)
		for i, e := range fi.failureEvents {
			if e.AgentID == agentID && e.JobIndex == jobIndex {
				fi.failureEvents[i].TimeRecovered = time.Now()
				fi.failureEvents[i].Recovered = true
			}
		}
		fi.failureEventsMu.Unlock()
	}()

	return FailureTypeTimeout
}

// injectCoordinatorCrash simulates coordinator crash and recovery
func (fi *FailureInjector) injectCoordinatorCrash(agentID int, jobIndex int) FailureType {
	event := FailureEvent{
		AgentID:      agentID,
		JobIndex:     jobIndex,
		FailureType:  "crash",
		TimeInjected: time.Now(),
	}

	fi.failureEventsMu.Lock()
	defer fi.failureEventsMu.Unlock()
	fi.failureEvents = append(fi.failureEvents, event)

	// Simulate: coordinator crashes, restarts, recovers jobs from DB
	go func() {
		// Simulate crash/restart delay
		time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)

		fi.failureEventsMu.Lock()
		// Mark as recovered (job resumed from checkpoint)
		for i, e := range fi.failureEvents {
			if e.AgentID == agentID && e.JobIndex == jobIndex {
				fi.failureEvents[i].TimeRecovered = time.Now()
				fi.failureEvents[i].Recovered = true
			}
		}
		fi.failureEventsMu.Unlock()
	}()

	return FailureTypeCrash
}

// RecoveryStats calculates recovery statistics
func (fi *FailureInjector) RecoveryStats() (injected int, recovered int, avgRecoveryMs float64) {
	fi.failureEventsMu.RLock()
	defer fi.failureEventsMu.RUnlock()

	injected = len(fi.failureEvents)
	var totalRecoveryTime time.Duration

	for _, event := range fi.failureEvents {
		if event.Recovered {
			recovered++
			totalRecoveryTime += event.TimeRecovered.Sub(event.TimeInjected)
		}
	}

	if recovered > 0 {
		avgRecoveryMs = float64(totalRecoveryTime.Milliseconds()) / float64(recovered)
	}

	return
}

// FailureType represents a type of failure
type FailureType int

const (
	FailureTypeNone FailureType = iota
	FailureTypeDisconnect
	FailureTypeTimeout
	FailureTypeCrash
	FailureTypeResourceExhaustion
)

// String returns the string representation of a failure type
func (ft FailureType) String() string {
	switch ft {
	case FailureTypeDisconnect:
		return "disconnect"
	case FailureTypeTimeout:
		return "timeout"
	case FailureTypeCrash:
		return "crash"
	case FailureTypeResourceExhaustion:
		return "resource_exhaustion"
	default:
		return "none"
	}
}
