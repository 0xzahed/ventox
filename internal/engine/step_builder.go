package engine

import "fmt"

// stepBuilder is a Builder for the ordered, human-readable step list of a plan.
// It keeps conditional step assembly declarative and out of the strategies' control flow.
type stepBuilder struct {
	steps []string
}

func newStepBuilder(initial ...string) *stepBuilder {
	return &stepBuilder{steps: append([]string(nil), initial...)}
}

// Add appends one or more steps.
func (b *stepBuilder) Add(steps ...string) *stepBuilder {
	b.steps = append(b.steps, steps...)
	return b
}

// Addf appends a single formatted step.
func (b *stepBuilder) Addf(format string, args ...any) *stepBuilder {
	return b.Add(fmt.Sprintf(format, args...))
}

// AddIf appends the steps only when cond is true.
func (b *stepBuilder) AddIf(cond bool, steps ...string) *stepBuilder {
	if cond {
		b.Add(steps...)
	}
	return b
}

// Build returns the accumulated steps.
func (b *stepBuilder) Build() []string {
	return b.steps
}
