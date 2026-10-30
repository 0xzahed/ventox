package engine

import (
	"sync"
	"time"
)

// ProgressEvent represents real-time progress of an active disk operation.
type ProgressEvent struct {
	StepIndex      int
	StepName       string
	BytesProcessed int64
	TotalBytes     int64
	Percent        float64
	SpeedBytesSec  float64
	ETA            time.Duration
}

// ProgressObserver is the Observer interface notified of progress events.
type ProgressObserver interface {
	OnProgress(event ProgressEvent)
}

// ProgressReporter is a function-typed ProgressObserver.
type ProgressReporter func(event ProgressEvent)

// OnProgress implements ProgressObserver.
func (r ProgressReporter) OnProgress(event ProgressEvent) {
	if r != nil {
		r(event)
	}
}

// ProgressBroadcaster is the Subject that fans events out to subscribed observers.
// It is safe for concurrent use.
type ProgressBroadcaster struct {
	mu        sync.RWMutex
	observers []ProgressObserver
}

// Subscribe registers an observer; nil observers are ignored.
func (b *ProgressBroadcaster) Subscribe(o ProgressObserver) {
	if o == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.observers = append(b.observers, o)
}

// OnProgress implements ProgressObserver by notifying every subscriber.
func (b *ProgressBroadcaster) OnProgress(event ProgressEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, o := range b.observers {
		o.OnProgress(event)
	}
}

// percentOf returns done/total as a percentage, or 0 when total is unknown.
func percentOf(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) / float64(total) * 100.0
}

// transferCallback adapts a byte-transfer callback (as used by the writer package)
// into progress events published to the observer for a given journal step.
func transferCallback(obs ProgressObserver, stepIndex int, stepName string) func(done, total int64, speed float64, eta time.Duration) {
	return func(done, total int64, speed float64, eta time.Duration) {
		if obs == nil {
			return
		}
		obs.OnProgress(ProgressEvent{
			StepIndex:      stepIndex,
			StepName:       stepName,
			BytesProcessed: done,
			TotalBytes:     total,
			Percent:        percentOf(done, total),
			SpeedBytesSec:  speed,
			ETA:            eta,
		})
	}
}
