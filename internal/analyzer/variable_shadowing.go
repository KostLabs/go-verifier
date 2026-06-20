package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
	"goverifier/internal/runner"
)

// VariableShadowing detects variables declared in inner scopes that shadow
// an outer variable with the same name, which can cause subtle bugs.
type VariableShadowing struct{}

func (VariableShadowing) Name() string {
	return "variable-shadowing"
}

func (VariableShadowing) Run(pass *runner.Pass) []report.Diagnostic {
	if pass.TypesInfo == nil {
		return nil
	}
	var diags []report.Diagnostic
	checkShadowing(pass, pass.File, pass.TypesInfo, &diags)
	return diags
}

func checkShadowing(pass *runner.Pass, file *ast.File, info *types.Info, diags *[]report.Diagnostic) {
	// Build a map from scope to its parent chain for quick ancestor lookup.
	// We use the types.Scope tree directly.

	// Collect all short variable declarations (:=) and check if any declared
	// name already exists in an enclosing scope.
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE {
			return true
		}
		if ignore.IsSuppressed(pass.IgnoreSet, assign.Pos(), "variable-shadowing") {
			return true
		}

		for _, lhs := range assign.Lhs {
			ident, isIdent := lhs.(*ast.Ident)
			if !isIdent || ident.Name == "_" {
				continue
			}
			obj := info.Defs[ident]
			if obj == nil {
				continue
			}
			innerScope := obj.Parent()
			if innerScope == nil {
				continue
			}
			// Walk up the scope chain looking for the same name.
			outer := innerScope.Parent()
			for outer != nil {
				if outerObj := outer.Lookup(ident.Name); outerObj != nil {
					// Found a shadowed variable.
					outerPos := pass.Fset.Position(outerObj.Pos())
					*diags = append(*diags, report.Diagnostic{
						Pos:     pass.Fset.Position(ident.Pos()),
						Rule:    "variable-shadowing",
						Message: "\"" + ident.Name + "\" shadows outer variable declared at " + outerPos.String(),
					})
					break
				}
				outer = outer.Parent()
			}
		}
		return true
	})
}
