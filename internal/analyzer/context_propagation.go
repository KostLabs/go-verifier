package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// ContextPropagation checks that functions accepting or performing I/O receive
// context.Context as their first parameter.
type ContextPropagation struct{}

func (ContextPropagation) Name() string { return "context-propagation" }

func (ContextPropagation) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			return true
		}

		if ignore.IsSuppressed(pass.IgnoreSet, fn.Pos(), "context-propagation") {
			return true
		}

		params := fn.Type.Params.List
		if len(params) == 0 {
			return true
		}

		// Check if the function already has context as first param.
		if isContextType(pass.TypesInfo, params[0].Type) {
			return true
		}

		// Flag only if any parameter or return value involves context,
		// or if the body contains calls that typically need context.
		if funcNeedsContext(pass.TypesInfo, fn) {
			pos := pass.Fset.Position(fn.Pos())
			diags = append(diags, report.Diagnostic{
				Pos:     pos,
				Rule:    "context-propagation",
				Message: "function " + fn.Name.Name + " should accept context.Context as its first parameter",
			})
		}

		return true
	})

	return diags
}

// isContextType reports whether expr resolves to a type that implements context.Context.
func isContextType(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	if typ := info.TypeOf(expr); typ != nil {
		return implementsContext(typ)
	}
	return false
}

// implementsContext reports whether typ implements context.Context, either by
// being context.Context itself or by satisfying its method set (e.g. *gin.Context).
func implementsContext(typ types.Type) bool {
	// Check for the exact context.Context named type.
	if named, ok := typ.(*types.Named); ok {
		if obj := named.Obj(); obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context" {
			return true
		}
	}
	// Check structural implementation: must have Deadline, Done, Err, Value methods.
	ms := types.NewMethodSet(typ)
	required := []string{"Deadline", "Done", "Err", "Value"}
	for _, name := range required {
		if ms.Lookup(nil, name) == nil {
			return false
		}
	}
	return true
}

// funcNeedsContext is a heuristic: a function needs context if any of its
// parameters are context.Context (already passed deeper) or if it calls
// functions that accept context (detected via selector expressions).
func funcNeedsContext(info *types.Info, fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if info == nil {
			return true
		}
		for _, arg := range call.Args {
			if typ := info.TypeOf(arg); typ != nil && implementsContext(typ) {
				found = true
				return false
			}
		}
		if sig := calleeSignature(info, call); sig != nil && sig.Params().Len() > 0 {
			if implementsContext(sig.Params().At(0).Type()) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// calleeSignature returns the *types.Signature for the function being called, if resolvable.
func calleeSignature(info *types.Info, call *ast.CallExpr) *types.Signature {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		if obj := info.ObjectOf(fn); obj != nil {
			sig, _ := obj.Type().(*types.Signature)
			return sig
		}
	case *ast.SelectorExpr:
		if sel := info.Selections[fn]; sel != nil {
			sig, _ := sel.Type().(*types.Signature)
			return sig
		}
		if obj := info.ObjectOf(fn.Sel); obj != nil {
			sig, _ := obj.Type().(*types.Signature)
			return sig
		}
	}
	return nil
}
