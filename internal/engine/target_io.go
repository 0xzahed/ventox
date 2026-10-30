package engine

import (
	"fmt"
	"io"
	"os"

	"github.com/0xzahed/ventox/internal/domain"
)

// TargetIO abstracts how the executor opens the destination device, letting tests
// and alternative back-ends substitute storage (Dependency Inversion).
type TargetIO interface {
	OpenWriter(plan *domain.WritePlan) (io.Writer, io.Closer, error)
	OpenReader(plan *domain.WritePlan) (io.Reader, io.Closer, error)
}

// deviceTargetIO opens the real block device, with optional per-direction overrides.
type deviceTargetIO struct {
	writer io.Writer
	reader io.Reader
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

func devicePath(plan *domain.WritePlan) string {
	if plan.Target.RawPath != "" {
		return plan.Target.RawPath
	}
	return plan.Target.Path
}

func (t deviceTargetIO) OpenWriter(plan *domain.WritePlan) (io.Writer, io.Closer, error) {
	if t.writer != nil {
		return t.writer, nopCloser{}, nil
	}
	f, err := os.OpenFile(devicePath(plan), os.O_WRONLY|os.O_SYNC, 0o666)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open target device for write: %w", err)
	}
	return f, f, nil
}

func (t deviceTargetIO) OpenReader(plan *domain.WritePlan) (io.Reader, io.Closer, error) {
	if t.reader != nil {
		if seeker, ok := t.reader.(io.Seeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return nil, nil, fmt.Errorf("failed to rewind target for verification: %w", err)
			}
		}
		return t.reader, nopCloser{}, nil
	}
	f, err := os.Open(devicePath(plan))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open target for verification: %w", err)
	}
	return f, f, nil
}
