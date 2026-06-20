package analyzer

import (
	"testing"
)

func TestDeferInLoop(t *testing.T) {
	a := DeferInLoop{}

	tests := []struct {
		name      string
		given     string
		src       string
		wantRules []string
	}{
		{
			name:  "defer inside range loop",
			given: "a range loop that defers file.Close() on each iteration",
			src: `package p
import "os"
func ProcessFiles(paths []string) error {
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	return nil
}`,
			wantRules: []string{"defer-in-loop"},
		},
		{
			name:  "defer inside classic for loop",
			given: "a classic for loop that defers a closure on each iteration",
			src: `package p
func CountDown(n int) {
	for i := n; i > 0; i-- {
		defer func(v int) { _ = v }(i)
	}
}`,
			wantRules: []string{"defer-in-loop"},
		},
		{
			name:  "defer inside function literal in loop does not trigger",
			given: "a range loop that immediately invokes a closure containing defer",
			src: `package p
import "os"
func ProcessFiles(paths []string) error {
	for _, path := range paths {
		func() {
			f, _ := os.Open(path)
			defer f.Close()
		}()
	}
	return nil
}`,
			wantRules: nil,
		},
		{
			name:  "defer outside any loop is fine",
			given: "a function with a top-level defer but no loop",
			src: `package p
import "os"
func ReadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}`,
			wantRules: nil,
		},
		{
			name:  "ignore directive suppresses defer-in-loop finding",
			given: "a range loop defer annotated with //goverifier:ignore:defer-in-loop",
			src: `package p
import "os"
func ProcessFiles(paths []string) error {
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		//goverifier:ignore:defer-in-loop
		defer f.Close()
	}
	return nil
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
