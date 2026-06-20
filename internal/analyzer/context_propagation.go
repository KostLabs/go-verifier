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
		first := params[0]
		if isContextType(pass.TypesInfo, first.Type) {
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
	t := info.TypeOf(expr)
	if t == nil {
		return false
	}
	return implementsContext(t)
}

// implementsContext reports whether t implements context.Context, either by
// being context.Context itself or by satisfying its method set (e.g. *gin.Context).
func implementsContext(t types.Type) bool {
	// Check for the exact context.Context named type.
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context" {
			return true
		}
	}
	// Check structural implementation: must have Deadline, Done, Err, Value methods.
	ms := types.NewMethodSet(t)
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
		// If any argument to a call is a context, the caller should accept one.
		for _, arg := range call.Args {
			if info != nil {
				t := info.TypeOf(arg)
				if t != nil && implementsContext(t) {
					found = true
					return false
				}
			}
		}
		// If the first arg type of the callee is context.Context, we should propagate.
		if info != nil {
			var fnType *types.Signature
			switch f := call.Fun.(type) {
			case *ast.Ident:
				if obj := info.ObjectOf(f); obj != nil {
					if sig, ok2 := obj.Type().(*types.Signature); ok2 {
						fnType = sig
					}
				}
			case *ast.SelectorExpr:
				if sel := info.Selections[f]; sel != nil {
					if sig, ok2 := sel.Type().(*types.Signature); ok2 {
						fnType = sig
					}
				} else if obj := info.ObjectOf(f.Sel); obj != nil {
					if sig, ok2 := obj.Type().(*types.Signature); ok2 {
						fnType = sig
					}
				}
			}
			if fnType != nil && fnType.Params().Len() > 0 {
				first := fnType.Params().At(0)
				if implementsContext(first.Type()) {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}
