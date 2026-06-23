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
//goverifier:ignore:dependency-inversion
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

// Write prints diagnostics to writer in the chosen format.
func Write(writer io.Writer, diags []Diagnostic, format Format) error {
	sort.Slice(diags, func(i, j int) bool {
		left, right := diags[i], diags[j]
		if left.Pos.Filename != right.Pos.Filename {
			return left.Pos.Filename < right.Pos.Filename
		}
		if left.Pos.Line != right.Pos.Line {
			return left.Pos.Line < right.Pos.Line
		}
		return left.Pos.Column < right.Pos.Column
	})

	switch format {
	case FormatJSON:
		return writeJSON(writer, diags)
	default:
		return writeText(writer, diags)
	}
}

func writeText(writer io.Writer, diags []Diagnostic) error {
	for _, diag := range diags {
		if _, err := fmt.Fprintf(writer, "%s:%d:%d: [%s] %s\n",
			diag.Pos.Filename, diag.Pos.Line, diag.Pos.Column, diag.Rule, diag.Message); err != nil {
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

func writeJSON(writer io.Writer, diags []Diagnostic) error {
	out := make([]jsonDiagnostic, len(diags))
	for i, diag := range diags {
		out[i] = jsonDiagnostic{
			File:    diag.Pos.Filename,
			Line:    diag.Pos.Line,
			Column:  diag.Pos.Column,
			Rule:    diag.Rule,
			Message: diag.Message,
		}
	}
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
