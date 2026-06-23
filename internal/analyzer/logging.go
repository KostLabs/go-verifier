package analyzer

import (
	"go/ast"
	"strings"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// Logging checks for bare fmt.Println / fmt.Printf / log.Print* calls in
// non-test production code. Structured logging (golog) should be used instead.
type Logging struct{}

func (Logging) Name() string { return "logging" }

// bannedCalls maps package name → set of banned function names.
var bannedCalls = map[string]map[string]bool{
	"fmt": {
		"Print":   true,
		"Println": true,
		"Printf":  true,
	},
	"log": {
		"Print":   true,
		"Println": true,
		"Printf":  true,
		"Fatal":   true,
		"Fatalf":  true,
		"Fatalln": true,
		"Panic":   true,
		"Panicf":  true,
		"Panicln": true,
	},
}

func (Logging) Run(pass *runner.Pass) []report.Diagnostic {
	if strings.HasSuffix(pass.Fset.File(pass.File.Pos()).Name(), "_test.go") {
		return nil
	}

	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ignore.IsSuppressed(pass.IgnoreSet, call.Pos(), "logging") {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if fns, found := bannedCalls[pkg.Name]; found && fns[sel.Sel.Name] {
			diags = append(diags, report.Diagnostic{
				Pos:     pass.Fset.Position(call.Pos()),
				Rule:    "logging",
				Message: pkg.Name + "." + sel.Sel.Name + " should not be used in production; use a structured logger (golog) instead",
			})
		}
		return true
	})

	return diags
}
