package analyzer

import (
	"testing"
)

func TestNakedReturn(t *testing.T) {
	a := NakedReturn{}

	tests := []struct {
		name      string
		given     string
		src       string
		wantRules []string
	}{
		{
			name:  "single naked return in named-result function",
			given: "a function with named results that uses a bare return",
			src: `package p
import "errors"
func Divide(a, b float64) (result float64, err error) {
	if b == 0 {
		err = errors.New("division by zero")
		return
	}
	result = a / b
	return
}`,
			wantRules: []string{"naked-return", "naked-return"},
		},
		{
			name:  "multiple naked returns across multiple named-result functions",
			given: "two functions each containing naked returns",
			src: `package p
import "errors"
func First() (val string, err error) {
	err = errors.New("oops")
	return
}
func Second() (n int, err error) {
	n = 1
	return
}`,
			wantRules: []string{"naked-return", "naked-return"},
		},
		{
			name:  "explicit return in named-result function is fine",
			given: "a function with named results that always uses explicit return values",
			src: `package p
import "errors"
func Divide(a, b float64) (result float64, err error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}`,
			wantRules: nil,
		},
		{
			name:  "unnamed results with empty return is fine",
			given: "a function with no named results that returns early",
			src: `package p
func DoNothing() {
	return
}`,
			wantRules: nil,
		},
		{
			name:  "ignore directive suppresses naked-return finding",
			given: "a function with naked return annotated with //goverifier:ignore:naked-return",
			src: `package p
import "errors"
//goverifier:ignore:naked-return
func Divide(a, b float64) (result float64, err error) {
	if b == 0 {
		err = errors.New("division by zero")
		return
	}
	result = a / b
	return
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
