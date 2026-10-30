package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/0xzahed/ventox/internal/domain"
)

type recordingHandler struct{ called bool }

func (h *recordingHandler) Execute(context.Context, *run) error {
	h.called = true
	return nil
}

func TestSelectStrategy(t *testing.T) {
	win := &domain.ImageMetadata{Classification: domain.ClassWindowsInstaller}
	cases := []struct {
		name string
		opts PlanOptions
		want planStrategy
	}{
		{"no source", PlanOptions{}, formatOnlyStrategy{}},
		{"windows", PlanOptions{Source: win}, windowsInstallerStrategy{}},
		{"linux", PlanOptions{Source: &domain.ImageMetadata{Classification: domain.ClassLinuxHybrid}}, linuxRawStrategy{}},
		{"unknown", PlanOptions{Source: &domain.ImageMetadata{}}, rawImageStrategy{}},
	}
	for _, tc := range cases {
		if got := selectStrategy(tc.opts); got != tc.want {
			t.Errorf("%s: got %T, want %T", tc.name, got, tc.want)
		}
	}
}

func TestStepBuilderAddIf(t *testing.T) {
	got := newStepBuilder("a").AddIf(false, "x").AddIf(true, "b").Addf("c%d", 1).Build()
	if len(got) != 3 || got[1] != "b" || got[2] != "c1" {
		t.Errorf("unexpected steps: %v", got)
	}
}

func TestProgressBroadcasterFansOut(t *testing.T) {
	var b ProgressBroadcaster
	var a, c int
	b.Subscribe(ProgressReporter(func(ProgressEvent) { a++ }))
	b.Subscribe(ProgressReporter(func(ProgressEvent) { c++ }))
	b.Subscribe(nil)
	b.OnProgress(ProgressEvent{})
	if a != 1 || c != 1 {
		t.Errorf("observers not notified: %d %d", a, c)
	}
}

func TestExecuteNilPlan(t *testing.T) {
	_, err := NewExecutor().Execute(context.Background(), nil, nil)
	if !errors.Is(err, domain.ErrValidationFailed) {
		t.Errorf("want ErrValidationFailed, got %v", err)
	}
}

func TestExecuteCancelledMapsToErrCancelled(t *testing.T) {
	plan, err := CreatePlan(PlanOptions{Target: domain.BlockDevice{Path: "/dev/x", SizeBytes: 1 << 30}, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	journal, err := NewExecutor().Execute(ctx, plan, nil)
	if !errors.Is(err, domain.ErrCancelled) || !errors.Is(err, context.Canceled) {
		t.Errorf("want ErrCancelled wrapping context.Canceled, got %v", err)
	}
	if journal == nil || journal.Err == nil {
		t.Errorf("journal should record the failure")
	}
}

func TestWithModeHandlerOverridesDefault(t *testing.T) {
	h := &recordingHandler{}
	plan, _ := CreatePlan(PlanOptions{Target: domain.BlockDevice{Path: "/dev/x", SizeBytes: 1 << 30}})
	e := NewExecutor(WithProvider(nil), WithTargetIO(discard{}, nil), WithModeHandler(domain.ModeFormatOnly, h))
	if _, err := e.Execute(context.Background(), plan, nil); err != nil {
		t.Fatal(err)
	}
	if !h.called {
		t.Errorf("custom handler was not invoked")
	}
}

func TestUnknownModeIsValidationError(t *testing.T) {
	plan, _ := CreatePlan(PlanOptions{Target: domain.BlockDevice{Path: "/dev/x", SizeBytes: 1 << 30}})
	plan.Mode = "Bogus"
	e := NewExecutor(WithProvider(nil), WithTargetIO(discard{}, nil))
	if _, err := e.Execute(context.Background(), plan, nil); !errors.Is(err, domain.ErrValidationFailed) {
		t.Errorf("want ErrValidationFailed, got %v", err)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
