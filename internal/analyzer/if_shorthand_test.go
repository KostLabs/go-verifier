package analyzer

import (
	"testing"
)

func TestIfShorthand(t *testing.T) {
	a := IfShorthand{}

	tests := []struct {
		name      string
		given     string
		src       string
		wantRules []string
	}{
		{
			name:  "simple err := ...; if err != nil",
			given: "assignment immediately before if that checks it, var not used after",
			src: `package p
func Do() error {
	err := run()
	if err != nil {
		return err
	}
	return nil
}
func run() error { return nil }`,
			wantRules: []string{"if-shorthand"},
		},
		{
			name:  "single value assignment used in condition",
			given: "non-error variable assigned then checked in if",
			src: `package p
func Check(n int) bool {
	v := compute(n)
	if v > 0 {
		return true
	}
	return false
}
func compute(n int) int { return n * 2 }`,
			wantRules: []string{"if-shorthand"},
		},
		{
			name:  "variable used after if — no flag",
			given: "assigned variable is referenced after the if block",
			src: `package p
func Do() error {
	err := run()
	if err != nil {
		return err
	}
	_ = err
	return nil
}
func run() error { return nil }`,
			wantRules: nil,
		},
		{
			name:  "if already has init statement — no flag",
			given: "the if statement already uses an init expression",
			src: `package p
func Do() error {
	if err := run(); err != nil {
		return err
	}
	return nil
}
func run() error { return nil }`,
			wantRules: nil,
		},
		{
			name:  "non-adjacent assignment — no flag",
			given: "a statement exists between the assignment and the if",
			src: `package p
func Do() error {
	err := run()
	doSomething()
	if err != nil {
		return err
	}
	return nil
}
func run() error { return nil }
func doSomething() {}`,
			wantRules: nil,
		},
		{
			name:  "assigned var not referenced in condition — no flag",
			given: "the if condition does not mention the declared variable",
			src: `package p
func Do(b bool) error {
	err := run()
	_ = err
	if b {
		return nil
	}
	return nil
}
func run() error { return nil }`,
			wantRules: nil,
		},
		{
			name:  "ignore directive on assignment suppresses finding",
			given: "//goverifier:ignore:if-shorthand placed before the assignment",
			src: `package p
func Do() error {
	//goverifier:ignore:if-shorthand
	err := run()
	if err != nil {
		return err
	}
	return nil
}
func run() error { return nil }`,
			wantRules: nil,
		},
		{
			name:  "multi-value assignment all used in condition",
			given: "two values declared and both appear in the if condition",
			src: `package p
func Do() bool {
	a, b := pair()
	if a > b {
		return true
	}
	return false
}
func pair() (int, int) { return 1, 2 }`,
			wantRules: []string{"if-shorthand"},
		},
		{
			name:  "multi-value assignment only partially used — no flag",
			given: "second declared var does not appear in the if condition",
			src: `package p
func Do() bool {
	a, b := pair()
	_ = b
	if a > 0 {
		return true
	}
	return false
}
func pair() (int, int) { return 1, 2 }`,
			wantRules: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runAnalyzer(t, a, "p", tc.src)
			assertDiags(t, got, tc.wantRules...)
		})
	}
}
