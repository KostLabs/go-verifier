package analyzer

import "testing"

func TestDependencyInversion(t *testing.T) {
	a := DependencyInversion{}

	tests := []struct {
		name      string
		src       string
		wantRules []string
	}{
		{
			name: "exported struct field with concrete stdlib type is flagged",
			src: `package p
import "net/http"
type Server struct {
	Client *http.Client
}`,
			wantRules: []string{"dependency-inversion"},
		},
		{
			name: "constructor accepting concrete stdlib type is flagged",
			src: `package p
import "net/http"
type Server struct{}
func NewServer(client *http.Client) *Server {
	return &Server{}
}`,
			wantRules: []string{"dependency-inversion"},
		},
		{
			name: "exported struct field with interface type is not flagged",
			src: `package p
import "io"
type Server struct {
	Writer io.Writer
}`,
			wantRules: nil,
		},
		{
			name: "constructor accepting interface type is not flagged",
			src: `package p
import "io"
type Server struct{}
func NewServer(w io.Writer) *Server {
	return &Server{}
}`,
			wantRules: nil,
		},
		{
			name: "unexported struct field with concrete type is not flagged",
			src: `package p
import "net/http"
type Server struct {
	client *http.Client
}`,
			wantRules: nil,
		},
		{
			name: "non-constructor function with concrete param is not flagged",
			src: `package p
import "net/http"
func DoSomething(client *http.Client) {}`,
			wantRules: nil,
		},
		{
			name: "ignore directive on constructor suppresses the finding",
			src: `package p
import "net/http"
type Server struct{}
//goverifier:ignore:dependency-inversion
func NewServer(client *http.Client) *Server {
	return &Server{}
}`,
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
