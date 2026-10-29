package engine

import "time"

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

// ProgressReporter receives progress updates during an operation.
type ProgressReporter func(event ProgressEvent)
