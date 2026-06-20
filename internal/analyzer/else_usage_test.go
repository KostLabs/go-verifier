package analyzer

import (
	"testing"
)

func TestElseUsage(t *testing.T) {
	a := ElseUsage{}

	tests := []struct {
		name      string
		given     string
		src       string
		wantRules []string
	}{
		{
			name:  "else after if that always returns",
			given: "an if block that returns followed by an unnecessary else",
			src: `package p
import "errors"
func Validate(name string) error {
	if name == "" {
		return errors.New("name is required")
	} else {
		return nil
	}
}`,
			wantRules: []string{"else-usage"},
		},
		{
			name:  "else after if that panics",
			given: "an if block that panics followed by an unnecessary else",
			src: `package p
func MustBePositive(n int) int {
	if n <= 0 {
		panic("n must be positive")
	} else {
		return n * 2
	}
}`,
			wantRules: []string{"else-usage"},
		},
		{
			name:  "nested else chains both flagged",
			given: "nested if/else where every if block exits",
			src: `package p
import "errors"
func Validate(name string) error {
	if name == "" {
		return errors.New("required")
	} else {
		if len(name) > 100 {
			return errors.New("too long")
		} else {
			return nil
		}
	}
}`,
			wantRules: []string{"else-usage", "else-usage"},
		},
		{
			name:  "else after if that does not always exit is fine",
			given: "an if block that may not exit (no return on all paths)",
			src: `package p
func Process(n int) int {
	if n > 0 {
		n = n * 2
	} else {
		n = 0
	}
	return n
}`,
			wantRules: nil,
		},
		{
			name:  "if without else is always fine",
			given: "an if block with no else clause",
			src: `package p
import "errors"
func Validate(name string) error {
	if name == "" {
		return errors.New("required")
	}
	return nil
}`,
			wantRules: nil,
		},
		{
			name:  "ignore directive suppresses else-usage finding",
			given: "an if/else where the if always returns, with //goverifier:ignore:else-usage on the if statement",
			src: `package p
import "errors"
func Validate(name string) error {
	//goverifier:ignore:else-usage
	if name == "" {
		return errors.New("name is required")
	} else {
		return nil
	}
}`,
			wantRules: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// When: the analyzer runs on the given source
			got := runAnalyzer(t, a, "p", tc.src)

			// Then: diagnostics match expectations
			assertDiags(t, got, tc.wantRules...)
		})
	}
}
