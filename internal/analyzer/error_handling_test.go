package analyzer

import (
	"testing"
)

func TestErrorHandling(t *testing.T) {
	a := ErrorHandling{}

	tests := []struct {
		name      string
		given     string
		src       string
		wantRules []string
	}{
		{
			name:  "bare panic in production code",
			given: "a non-test function that calls panic instead of returning an error",
			src: `package p
func MustParse(s string) int {
	if s == "" {
		panic("empty string")
	}
	return 42
}`,
			wantRules: []string{"error-handling"},
		},
		{
			name:  "fmt.Errorf using %v instead of %w to wrap an error",
			given: "a fmt.Errorf call that uses %v when wrapping an error value",
			src: `package p
import (
	"errors"
	"fmt"
)
var ErrNotFound = errors.New("not found")
func FindUser(id int) error {
	return fmt.Errorf("finding user: %v", ErrNotFound)
}`,
			wantRules: []string{"error-handling"},
		},
		{
			name:  "error is not the last return value",
			given: "a function where error appears before a non-error return value",
			src: `package p
func LoadConfig() (error, string) {
	return nil, "config"
}`,
			wantRules: []string{"error-handling"},
		},
		{
			name:  "fmt.Errorf using %w wraps error correctly",
			given: "a fmt.Errorf call that uses %w to wrap an error",
			src: `package p
import (
	"errors"
	"fmt"
)
var ErrNotFound = errors.New("not found")
func FindUser(id int) error {
	return fmt.Errorf("finding user: %w", ErrNotFound)
}`,
			wantRules: nil,
		},
		{
			name:  "error is the last return value",
			given: "a function that correctly places error last in its return list",
			src: `package p
func LoadConfig() (string, error) {
	return "config", nil
}`,
			wantRules: nil,
		},
		{
			name:  "ignore directive suppresses panic finding",
			given: "a panic call annotated with //goverifier:ignore:error-handling",
			src: `package p
func MustParse(s string) int {
	if s == "" {
		//goverifier:ignore:error-handling
		panic("empty string")
	}
	return 42
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
