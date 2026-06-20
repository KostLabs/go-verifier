package analyzer

import (
	"testing"
)

func TestLogging(t *testing.T) {
	a := Logging{}

	tests := []struct {
		name      string
		given     string
		filename  string // set to override default; empty = "input.go"
		src       string
		wantRules []string
	}{
		{
			name:  "fmt.Println in production code",
			given: "a production function that calls fmt.Println",
			src: `package p
import "fmt"
func CreateUser(name string) {
	fmt.Println("creating user:", name)
}`,
			wantRules: []string{"logging"},
		},
		{
			name:  "fmt.Printf in production code",
			given: "a production function that calls fmt.Printf",
			src: `package p
import "fmt"
func CreateUser(name string) {
	fmt.Printf("user: %s\n", name)
}`,
			wantRules: []string{"logging"},
		},
		{
			name:  "log.Printf in production code",
			given: "a production function that calls log.Printf",
			src: `package p
import "log"
func DeleteUser(id int) {
	log.Printf("deleting user %d", id)
}`,
			wantRules: []string{"logging"},
		},
		{
			name:  "log.Fatal in production code",
			given: "a production function that calls log.Fatal",
			src: `package p
import "log"
func StartServer() {
	log.Fatal("failed to start")
}`,
			wantRules: []string{"logging"},
		},
		{
			name:  "multiple banned calls in one function",
			given: "a production function that calls both fmt.Println and log.Printf",
			src: `package p
import (
	"fmt"
	"log"
)
func Handle() {
	fmt.Println("start")
	log.Printf("also bad")
}`,
			wantRules: []string{"logging", "logging"},
		},
		{
			name:  "fmt.Println in test file is allowed",
			given: "a test file containing fmt.Println",
			src: `package p
import "fmt"
func TestHelper() {
	fmt.Println("debug output in test")
}`,
			wantRules: nil,
		},
		{
			name:  "ignore directive suppresses logging finding",
			given: "a fmt.Println call annotated with //goverifier:ignore:logging",
			src: `package p
import "fmt"
func CreateUser(name string) {
	//goverifier:ignore:logging
	fmt.Println("creating user:", name)
}`,
			wantRules: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The logging analyzer checks the filename for _test.go suffix.
			// We embed the filename decision in the source package name for the
			// test file case by using a test-suffixed file in runAnalyzerFile.
			got := runAnalyzer(t, a, "p", tc.src)
			if tc.name == "fmt.Println in test file is allowed" {
				got = runAnalyzerInTestFile(t, a, tc.src)
			}

			// Then: diagnostics match expectations
			assertDiags(t, got, tc.wantRules...)
		})
	}
}
