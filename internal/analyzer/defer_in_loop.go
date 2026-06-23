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

		switch node := n.(type) {
		case *ast.ForStmt:
			findDeferInLoop(pass, node.Body, true, diags)
			return false
		case *ast.RangeStmt:
			findDeferInLoop(pass, node.Body, true, diags)
			return false

		case *ast.FuncLit:
			// Reset loop context for inner functions.
			findDeferInLoop(pass, node.Body, false, diags)
			return false

		case *ast.DeferStmt:
			if inLoop && !ignore.IsSuppressed(pass.IgnoreSet, node.Pos(), "defer-in-loop") {
				*diags = append(*diags, report.Diagnostic{
					Pos:     pass.Fset.Position(node.Pos()),
					Rule:    "defer-in-loop",
					Message: "defer inside loop will not execute until the function returns; extract the loop body into a separate function",
				})
			}
		}
		return true
	})
}
