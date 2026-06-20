// Package report formats diagnostic output.
package report

import (
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"sort"
)

// Diagnostic is a single finding from an analyzer.
type Diagnostic struct {
	Pos     token.Position
	Rule    string
	Message string
}

// Format controls output style.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Write prints diagnostics to w in the chosen format.
func Write(w io.Writer, diags []Diagnostic, fmt_ Format) error {
	sort.Slice(diags, func(i, j int) bool {
		a, b := diags[i], diags[j]
		if a.Pos.Filename != b.Pos.Filename {
			return a.Pos.Filename < b.Pos.Filename
		}
		if a.Pos.Line != b.Pos.Line {
			return a.Pos.Line < b.Pos.Line
		}
		return a.Pos.Column < b.Pos.Column
	})

	switch fmt_ {
	case FormatJSON:
		return writeJSON(w, diags)
	default:
		return writeText(w, diags)
	}
}

func writeText(w io.Writer, diags []Diagnostic) error {
	for _, d := range diags {
		_, err := fmt.Fprintf(w, "%s:%d:%d: [%s] %s\n",
			d.Pos.Filename, d.Pos.Line, d.Pos.Column, d.Rule, d.Message)
		if err != nil {
			return err
		}
	}
	return nil
}

type jsonDiagnostic struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

func writeJSON(w io.Writer, diags []Diagnostic) error {
	out := make([]jsonDiagnostic, len(diags))
	for i, d := range diags {
		out[i] = jsonDiagnostic{
			File:    d.Pos.Filename,
			Line:    d.Pos.Line,
			Column:  d.Pos.Column,
			Rule:    d.Rule,
			Message: d.Message,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
