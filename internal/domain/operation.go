package domain

import "time"

type StepStatus string

const (
	StatusPending   StepStatus = "Pending"
	StatusRunning   StepStatus = "Running"
	StatusCompleted StepStatus = "Completed"
	StatusSkipped   StepStatus = "Skipped"
	StatusFailed    StepStatus = "Failed"
)

type JournalEntry struct {
	StepIndex int
	Name      string
	Status    StepStatus
	StartedAt time.Time
	Duration  time.Duration
	Message   string
}

type OperationJournal struct {
	PlanID    string
	StartedAt time.Time
	EndedAt   time.Time
	Entries   []JournalEntry
	Err       error
}

func (j *OperationJournal) AddEntry(name string) int {
	idx := len(j.Entries) + 1
	j.Entries = append(j.Entries, JournalEntry{
		StepIndex: idx,
		Name:      name,
		Status:    StatusPending,
		StartedAt: time.Now(),
	})
	return idx - 1
}

func (j *OperationJournal) CompleteEntry(idx int, status StepStatus, msg string) {
	if idx >= 0 && idx < len(j.Entries) {
		j.Entries[idx].Status = status
		j.Entries[idx].Message = msg
		j.Entries[idx].Duration = time.Since(j.Entries[idx].StartedAt)
	}
}

type VerificationReport struct {
	SourceDigest string
	TargetDigest string
	Verified     bool
	BytesChecked int64
	Duration     time.Duration
}
