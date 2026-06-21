package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// DependencyInversion flags exported struct fields and constructor function
// parameters that are concrete struct/pointer types instead of interfaces.
// High-level modules should depend on abstractions, not concretions.
//
// Heuristic:
//   - Exported struct fields whose type is a named struct or *struct from another
//     package are flagged (e.g. field DB *sql.DB should be an interface).
//   - Functions named New* whose parameters are concrete types are flagged.
//
// Standard library types that are universally used as concrete values
// (context.Context is an interface so it's fine; *sql.DB, *http.Client, etc.
// are flagged as intended).
type DependencyInversion struct{}

func (DependencyInversion) Name() string { return "dependency-inversion" }

func (DependencyInversion) Run(pass *runner.Pass) []report.Diagnostic {
	if pass.TypesInfo == nil {
		return nil
	}
	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			checkStructFields(pass, node, &diags)
		case *ast.FuncDecl:
			checkConstructorParams(pass, node, &diags)
		}
		return true
	})

	return diags
}

func checkStructFields(pass *runner.Pass, ts *ast.TypeSpec, diags *[]report.Diagnostic) {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return
	}
	if ignore.IsSuppressed(pass.IgnoreSet, ts.Pos(), "dependency-inversion") {
		return
	}

	for _, field := range st.Fields.List {
		// Only check exported fields (uppercase first letter).
		if len(field.Names) > 0 && !field.Names[0].IsExported() {
			continue
		}
		if ignore.IsSuppressed(pass.IgnoreSet, field.Pos(), "dependency-inversion") {
			continue
		}
		if isConcreteDependency(pass.TypesInfo, pass.Pkg, field.Type) {
			name := "<embedded>"
			if len(field.Names) > 0 {
				name = field.Names[0].Name
			}
			*diags = append(*diags, report.Diagnostic{
				Pos:     pass.Fset.Position(field.Pos()),
				Rule:    "dependency-inversion",
				Message: "field " + name + " uses a concrete type; prefer an interface to decouple the dependency",
			})
		}
	}
}

// checkConstructorParams flags New* functions whose parameters are concrete types.
func checkConstructorParams(pass *runner.Pass, fn *ast.FuncDecl, diags *[]report.Diagnostic) {
	if fn.Name == nil || len(fn.Name.Name) < 4 {
		return
	}
	if fn.Name.Name[:3] != "New" {
		return
	}
	if fn.Type.Params == nil {
		return
	}
	if ignore.IsSuppressed(pass.IgnoreSet, fn.Pos(), "dependency-inversion") {
		return
	}

	for _, param := range fn.Type.Params.List {
		if isConcreteDependency(pass.TypesInfo, pass.Pkg, param.Type) {
			name := "_"
			if len(param.Names) > 0 {
				name = param.Names[0].Name
			}
			*diags = append(*diags, report.Diagnostic{
				Pos:     pass.Fset.Position(param.Pos()),
				Rule:    "dependency-inversion",
				Message: "parameter " + name + " in constructor " + fn.Name.Name + " uses a concrete type; prefer an interface",
			})
		}
	}
}

// isConcreteDependency reports whether expr is a concrete struct or pointer-to-struct
// type from an external package (not a primitive, interface, or same-package type).
func isConcreteDependency(info *types.Info, currentPkg *types.Package, expr ast.Expr) bool {
	t := info.TypeOf(expr)
	if t == nil {
		return false
	}
	return isConcreteExternalType(t, currentPkg)
}

func isConcreteExternalType(t types.Type, currentPkg *types.Package) bool {
	switch typ := t.(type) {
	case *types.Pointer:
		return isConcreteExternalType(typ.Elem(), currentPkg)
	case *types.Named:
		obj := typ.Obj()
		if obj.Pkg() == nil {
			// Universe type (builtin).
			return false
		}

		// Same-package types are not cross-package dependencies — skip.
		if currentPkg != nil && obj.Pkg() == currentPkg {
			return false
		}

		// Check underlying is a struct, not an interface.
		switch typ.Underlying().(type) {
		case *types.Struct:
			return true
		case *types.Interface:
			// Already an interface — fine.
			return false
		}
	}
	return false
}
