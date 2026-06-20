package analyzer

import (
	"go/ast"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// DeferInLoop checks for defer statements inside for/range loop bodies.
// Deferred calls stack until the enclosing function returns, which can
// exhaust resources when the loop has many iterations.
type DeferInLoop struct{}

func (DeferInLoop) Name() string { return "defer-in-loop" }

func (DeferInLoop) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic
	findDeferInLoop(pass, pass.File, false, &diags)
	return diags
}

// findDeferInLoop walks the AST tracking whether we are currently inside a loop.
// It resets the loop flag when entering a new function literal so that defers
// inside an immediately-invoked closure inside a loop are fine.
func findDeferInLoop(pass *runner.Pass, node ast.Node, inLoop bool, diags *[]report.Diagnostic) {
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		switch x := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			// Recurse with inLoop=true.
			var body *ast.BlockStmt
			switch v := x.(type) {
			case *ast.ForStmt:
				body = v.Body
			case *ast.RangeStmt:
				body = v.Body
			}
			if body != nil {
				findDeferInLoop(pass, body, true, diags)
			}
			return false // already handled children

		case *ast.FuncLit:
			// Reset loop context for inner functions.
			findDeferInLoop(pass, x.Body, false, diags)
			return false

		case *ast.DeferStmt:
			if inLoop && !ignore.IsSuppressed(pass.IgnoreSet, x.Pos(), "defer-in-loop") {
				*diags = append(*diags, report.Diagnostic{
					Pos:     pass.Fset.Position(x.Pos()),
					Rule:    "defer-in-loop",
					Message: "defer inside loop will not execute until the function returns; extract the loop body into a separate function",
				})
			}
		}
		return true
	})
}
