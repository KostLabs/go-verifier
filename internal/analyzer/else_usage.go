package analyzer

import (
	"go/ast"
	"go/token"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
	"goverifier/internal/runner"
)

// ElseUsage flags else blocks that follow an if-body that unconditionally
// exits (return, panic, continue, break, goto). The else is redundant and
// violates the early-return / flatten-logic principle.
type ElseUsage struct{}

func (ElseUsage) Name() string { return "else-usage" }

func (ElseUsage) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if ifStmt.Else == nil {
			return true
		}
		if ignore.IsSuppressed(pass.IgnoreSet, ifStmt.Pos(), "else-usage") {
			return true
		}
		if blockAlwaysExits(ifStmt.Body) {
			diags = append(diags, report.Diagnostic{
				Pos:     pass.Fset.Position(ifStmt.Else.Pos()),
				Rule:    "else-usage",
				Message: "else block is unnecessary after an if block that always returns/exits; use early return instead",
			})
		}
		return true
	})

	return diags
}

// blockAlwaysExits reports whether every code path in b ends with an
// unconditional exit (return, panic, continue, break, goto).
func blockAlwaysExits(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	return stmtAlwaysExits(b.List[len(b.List)-1])
}

func stmtAlwaysExits(s ast.Stmt) bool {
	_ = token.NoPos
	switch v := s.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		// continue, break, goto, fallthrough
		return true
	case *ast.ExprStmt:
		// panic(...)
		call, ok := v.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		ident, ok := call.Fun.(*ast.Ident)
		return ok && ident.Name == "panic"
	case *ast.BlockStmt:
		return blockAlwaysExits(v)
	case *ast.IfStmt:
		// An if/else where both branches exit always exits.
		if v.Else == nil {
			return false
		}
		return blockAlwaysExits(v.Body) && stmtAlwaysExits(v.Else)
	case *ast.SwitchStmt:
		return switchAlwaysExits(v)
	}
	return false
}

func switchAlwaysExits(sw *ast.SwitchStmt) bool {
	if sw.Body == nil {
		return false
	}
	hasDefault := false
	for _, stmt := range sw.Body.List {
		cc, ok := stmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		if cc.List == nil {
			hasDefault = true
		}
		if len(cc.Body) == 0 {
			return false
		}
		if !stmtAlwaysExits(cc.Body[len(cc.Body)-1]) {
			return false
		}
	}
	return hasDefault
}
