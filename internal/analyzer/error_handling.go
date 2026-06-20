package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
	"goverifier/internal/runner"
)

// ErrorHandling checks:
//   - No bare panic() calls (except in init/main or test files)
//   - Error wrapping uses %w (not %s/%v) when fmt.Errorf is called
//   - error is the last return value in functions that return it
type ErrorHandling struct{}

func (ErrorHandling) Name() string { return "error-handling" }

func (ErrorHandling) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic
	filename := pass.Fset.File(pass.File.Pos()).Name()
	isTest := strings.HasSuffix(filename, "_test.go")

	ast.Inspect(pass.File, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if ignore.IsSuppressed(pass.IgnoreSet, node.Pos(), "error-handling") {
				return true
			}
			checkPanic(pass, node, isTest, &diags)
			checkErrorfWrapping(pass, node, &diags)

		case *ast.FuncDecl:
			if ignore.IsSuppressed(pass.IgnoreSet, node.Pos(), "error-handling") {
				return true
			}
			checkErrorLastReturn(pass, node, &diags)
		}
		return true
	})

	return diags
}

func checkPanic(pass *runner.Pass, call *ast.CallExpr, isTest bool, diags *[]report.Diagnostic) {
	if isTest {
		return
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "panic" {
		return
	}
	// Allow panic(err) where err is a builtin — e.g. panic("unreachable") in truly
	// unreachable branches is common. We flag all panics and let the ignore directive
	// handle justified cases.
	*diags = append(*diags, report.Diagnostic{
		Pos:     pass.Fset.Position(call.Pos()),
		Rule:    "error-handling",
		Message: "avoid panic; return an error instead",
	})
}

func checkErrorfWrapping(pass *runner.Pass, call *ast.CallExpr, diags *[]report.Diagnostic) {
	// Match fmt.Errorf(...)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Errorf" {
		return
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "fmt" {
		return
	}
	if len(call.Args) == 0 {
		return
	}
	// First arg must be a format string literal containing %w if wrapping an error.
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return
	}
	format := strings.Trim(lit.Value, `"`)

	// If there's an error argument but the format uses %v or %s instead of %w, flag it.
	if hasErrorArg(pass, call) && !strings.Contains(format, "%w") {
		*diags = append(*diags, report.Diagnostic{
			Pos:     pass.Fset.Position(call.Pos()),
			Rule:    "error-handling",
			Message: "use %w instead of %v or %s to wrap errors so errors.Is/As work correctly",
		})
	}
}

// hasErrorArg reports whether any argument to the call implements error.
func hasErrorArg(pass *runner.Pass, call *ast.CallExpr) bool {
	if pass.TypesInfo == nil {
		return false
	}
	errorType := types.Universe.Lookup("error").Type()
	for _, arg := range call.Args[1:] {
		t := pass.TypesInfo.TypeOf(arg)
		if t != nil && types.Implements(t, errorType.Underlying().(*types.Interface)) {
			return true
		}
	}
	return false
}

// checkErrorLastReturn flags functions where error is not the last return value.
func checkErrorLastReturn(pass *runner.Pass, fn *ast.FuncDecl, diags *[]report.Diagnostic) {
	if fn.Type.Results == nil {
		return
	}
	results := fn.Type.Results.List
	if len(results) < 2 {
		return
	}
	errorType := types.Universe.Lookup("error").Type()

	// Find any result that is error but is not the last one.
	for i, field := range results {
		if i == len(results)-1 {
			break
		}
		if pass.TypesInfo == nil {
			continue
		}
		t := pass.TypesInfo.TypeOf(field.Type)
		if t == nil {
			continue
		}
		if types.Identical(t, errorType) {
			*diags = append(*diags, report.Diagnostic{
				Pos:     pass.Fset.Position(fn.Pos()),
				Rule:    "error-handling",
				Message: "function " + fn.Name.Name + ": error should be the last return value",
			})
			// Only report once per function.
			return
		}
	}

	// Also check that the last return value isn't non-error while there's an
	// error somewhere non-last — already handled above.
	_ = token.NoPos
}
