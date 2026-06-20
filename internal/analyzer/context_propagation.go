package analyzer

import (
	"go/ast"
	"go/types"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
	"goverifier/internal/runner"
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

// isContextType reports whether expr resolves to context.Context.
func isContextType(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}
	t := info.TypeOf(expr)
	if t == nil {
		return false
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
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
				if t != nil {
					if named, ok2 := t.(*types.Named); ok2 {
						obj := named.Obj()
						if obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context" {
							found = true
							return false
						}
					}
				}
			}
			// Fallback: check if the ident is named "ctx" or "context".
			if ident, ok2 := arg.(*ast.Ident); ok2 {
				if ident.Name == "ctx" || ident.Name == "context" {
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
				if named, ok2 := first.Type().(*types.Named); ok2 {
					obj := named.Obj()
					if obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context" {
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}
