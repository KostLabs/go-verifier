package analyzer

import (
	"go/ast"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
	"goverifier/internal/runner"
)

// NakedReturn checks for functions that use named return values with bare
// "return" statements. Naked returns obscure what a function actually returns.
type NakedReturn struct{}

func (NakedReturn) Name() string { return "naked-return" }

func (NakedReturn) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if ignore.IsSuppressed(pass.IgnoreSet, fn.Pos(), "naked-return") {
			return true
		}
		if !hasNamedResults(fn) || fn.Body == nil {
			return true
		}
		// Walk body looking for bare return statements.
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			ret, isRet := inner.(*ast.ReturnStmt)
			if !isRet {
				return true
			}
			if len(ret.Results) == 0 {
				if !ignore.IsSuppressed(pass.IgnoreSet, ret.Pos(), "naked-return") {
					diags = append(diags, report.Diagnostic{
						Pos:     pass.Fset.Position(ret.Pos()),
						Rule:    "naked-return",
						Message: "naked return in function with named results; explicitly list return values",
					})
				}
			}
			return true
		})
		return true
	})

	return diags
}

func hasNamedResults(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, field := range fn.Type.Results.List {
		if len(field.Names) > 0 {
			return true
		}
	}
	return false
}
