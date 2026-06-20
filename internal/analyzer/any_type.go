package analyzer

import (
	"go/ast"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// AnyType flags uses of the `any` type alias and bare interface{} in
// positions where a concrete type could be used. This catches loss of
// type safety outside of marshalling/formatting contexts.
type AnyType struct{}

func (AnyType) Name() string { return "any-type" }

// exemptFunctions are contexts where any/interface{} is idiomatic.
var exemptFunctions = map[string]bool{
	"Marshal":       true,
	"Unmarshal":     true,
	"MarshalJSON":   true,
	"UnmarshalJSON": true,
	"Scan":          true,
	"Sprintf":       true,
	"Fprintf":       true,
	"Printf":        true,
	"Errorf":        true,
}

func (AnyType) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Field:
			if ignore.IsSuppressed(pass.IgnoreSet, node.Pos(), "any-type") {
				return true
			}
			if isAnyType(node.Type) {
				diags = append(diags, report.Diagnostic{
					Pos:     pass.Fset.Position(node.Pos()),
					Rule:    "any-type",
					Message: "avoid using any/interface{}; use a concrete type or a typed interface instead",
				})
			}

		case *ast.FuncDecl:
			if ignore.IsSuppressed(pass.IgnoreSet, node.Pos(), "any-type") {
				return true
			}
			// Skip exempt function names (marshalling etc.)
			if exemptFunctions[node.Name.Name] {
				return false
			}

		case *ast.ValueSpec:
			if ignore.IsSuppressed(pass.IgnoreSet, node.Pos(), "any-type") {
				return true
			}
			if node.Type != nil && isAnyType(node.Type) {
				diags = append(diags, report.Diagnostic{
					Pos:     pass.Fset.Position(node.Pos()),
					Rule:    "any-type",
					Message: "avoid using any/interface{} for variable type; use a concrete type instead",
				})
			}
		}
		return true
	})

	return diags
}

// isAnyType returns true for the `any` ident and bare empty interface{}.
func isAnyType(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name == "any"
	case *ast.InterfaceType:
		return e.Methods == nil || len(e.Methods.List) == 0
	}
	return false
}
