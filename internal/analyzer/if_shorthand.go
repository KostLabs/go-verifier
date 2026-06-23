package analyzer

import (
	"go/ast"
	"go/token"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// IfShorthand flags patterns where a variable is assigned immediately before
// an if-statement that checks it, and the variable is not used after the if.
// These can be collapsed into an if-init statement: if v := ...; v != nil { ... }
type IfShorthand struct{}

func (IfShorthand) Name() string { return "if-shorthand" }

func (IfShorthand) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		stmts := block.List
		for i := 0; i < len(stmts)-1; i++ {
			assign, isAssign := stmts[i].(*ast.AssignStmt)
			if !isAssign || assign.Tok != token.DEFINE {
				continue
			}
			ifStmt, isIf := stmts[i+1].(*ast.IfStmt)
			if !isIf || ifStmt.Init != nil {
				continue
			}
			if ignore.IsSuppressed(pass.IgnoreSet, assign.Pos(), "if-shorthand") {
				continue
			}
			if ignore.IsSuppressed(pass.IgnoreSet, ifStmt.Pos(), "if-shorthand") {
				continue
			}

			// Collect names declared by the assignment.
			declared := declaredNames(assign)
			if len(declared) == 0 {
				continue
			}

			// All declared names must appear in the if condition and not after the if.
			if !allUsedInCond(declared, ifStmt.Cond) {
				continue
			}
			if anyUsedAfter(declared, stmts[i+2:]) {
				continue
			}

			diags = append(diags, report.Diagnostic{
				Pos:     pass.Fset.Position(assign.Pos()),
				Rule:    "if-shorthand",
				Message: "assignment can be moved into the if init statement: if " + formatAssign(assign) + "; " + formatExpr(ifStmt.Cond) + " { ... }",
			})
		}
		return true
	})

	return diags
}

// declaredNames returns all names on the LHS of a := that are new declarations
// (i.e. not blank identifiers).
func declaredNames(assign *ast.AssignStmt) []string {
	var names []string
	for _, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		names = append(names, ident.Name)
	}
	return names
}

// allUsedInCond reports whether every name in names appears at least once in expr.
func allUsedInCond(names []string, expr ast.Expr) bool {
	used := make(map[string]bool, len(names))
	ast.Inspect(expr, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if ok {
			used[ident.Name] = true
		}
		return true
	})
	for _, name := range names {
		if !used[name] {
			return false
		}
	}
	return true
}

// anyUsedAfter reports whether any of the names appears in any of the remaining statements.
func anyUsedAfter(names []string, stmts []ast.Stmt) bool {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	for _, s := range stmts {
		found := false
		ast.Inspect(s, func(n ast.Node) bool {
			if found {
				return false
			}
			ident, ok := n.(*ast.Ident)
			found = ok && nameSet[ident.Name]
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

// formatAssign produces a compact representation of the assignment for the message.
func formatAssign(assign *ast.AssignStmt) string {
	lhs := ""
	for i, l := range assign.Lhs {
		if i > 0 {
			lhs += ", "
		}
		lhs += formatExpr(l)
	}
	rhs := ""
	for i, r := range assign.Rhs {
		if i > 0 {
			rhs += ", "
		}
		rhs += formatExpr(r)
	}
	return lhs + " := " + rhs
}

// formatExpr produces a short string for an expression node.
func formatExpr(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.CallExpr:
		return formatExpr(node.Fun) + "(...)"
	case *ast.SelectorExpr:
		return formatExpr(node.X) + "." + node.Sel.Name
	case *ast.BinaryExpr:
		return formatExpr(node.X) + " " + node.Op.String() + " " + formatExpr(node.Y)
	case *ast.UnaryExpr:
		return node.Op.String() + formatExpr(node.X)
	case *ast.BasicLit:
		return node.Value
	}
	return "..."
}
