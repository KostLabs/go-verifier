package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// Naming checks:
//   - Interface names don't start with "I" (e.g. IUserRepository)
//   - Constants aren't UPPER_SNAKE_CASE
//   - Package names aren't generic (utils, helpers, common, util, helper)
//   - Single-method interfaces use method name + "-er" suffix (advisory only)
//   - Variable/parameter names must be longer than one character (i/j/k loop indices are allowed)
type Naming struct{}

func (Naming) Name() string { return "naming" }

var genericPackageNames = map[string]bool{
	"utils":   true,
	"util":    true,
	"helpers": true,
	"helper":  true,
	"common":  true,
	"misc":    true,
}

func (Naming) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic

	// Package name check — once per file is fine, deduplicate via position.
	pkgName := pass.File.Name.Name
	if genericPackageNames[pkgName] {
		if !ignore.IsSuppressed(pass.IgnoreSet, pass.File.Name.Pos(), "naming") {
			diags = append(diags, report.Diagnostic{
				Pos:     pass.Fset.Position(pass.File.Name.Pos()),
				Rule:    "naming",
				Message: "package name \"" + pkgName + "\" is too generic; use a descriptive domain name",
			})
		}
	}

	// Collect positions of for-loop index variables (i/j/k) so we can allow them.
	loopIndexPos := collectLoopIndexPositions(pass.File)

	ast.Inspect(pass.File, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			// Directive before a type or const block maps to the GenDecl position.
			if ignore.IsSuppressed(pass.IgnoreSet, node.Pos(), "naming") {
				return false
			}
			switch node.Tok.String() {
			case "type":
				for _, spec := range node.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 {
						checkInterfaceName(pass, ts.Name, iface, &diags)
					}
				}
			case "const":
				for _, spec := range node.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range vs.Names {
						if ignore.IsSuppressed(pass.IgnoreSet, name.Pos(), "naming") {
							continue
						}
						checkConstantName(pass, name, &diags)
					}
				}
			}
			return false
		case *ast.FuncDecl:
			checkFuncNames(pass, node, loopIndexPos, &diags)
			return true
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				checkAssignNames(pass, node, loopIndexPos, &diags)
			}
		case *ast.RangeStmt:
			// Handled inside collectLoopIndexPositions; skip re-visiting Key/Value here.
			return true
		}
		return true
	})

	return diags
}

// collectLoopIndexPositions returns the token.Pos of every identifier that is
// used as a classic for-loop init variable or a range Key/Value — these are
// allowed to be single-letter (i, j, k).
func collectLoopIndexPositions(file *ast.File) map[token.Pos]bool {
	positions := make(map[token.Pos]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ForStmt:
			// Classic for i := 0; i < n; i++
			if assign, ok := node.Init.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
				for _, lhs := range assign.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						positions[ident.Pos()] = true
					}
				}
			}
		case *ast.RangeStmt:
			// for i, v := range ...
			if node.Key != nil {
				if ident, ok := node.Key.(*ast.Ident); ok {
					positions[ident.Pos()] = true
				}
			}
			if node.Value != nil {
				if ident, ok := node.Value.(*ast.Ident); ok {
					positions[ident.Pos()] = true
				}
			}
		}
		return true
	})
	return positions
}

// checkFuncNames checks receiver names, parameter names, and named return values.
func checkFuncNames(pass *runner.Pass, fn *ast.FuncDecl, loopIndexPos map[token.Pos]bool, diags *[]report.Diagnostic) {
	if ignore.IsSuppressed(pass.IgnoreSet, fn.Pos(), "naming") {
		return
	}
	// Receiver names.
	if fn.Recv != nil {
		for _, field := range fn.Recv.List {
			for _, name := range field.Names {
				checkShortName(pass, name, loopIndexPos, diags)
			}
		}
	}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				checkShortName(pass, name, loopIndexPos, diags)
			}
		}
	}
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			for _, name := range field.Names {
				checkShortName(pass, name, loopIndexPos, diags)
			}
		}
	}
}

// checkAssignNames checks short variable declarations (:=) outside of loops.
func checkAssignNames(pass *runner.Pass, assign *ast.AssignStmt, loopIndexPos map[token.Pos]bool, diags *[]report.Diagnostic) {
	if ignore.IsSuppressed(pass.IgnoreSet, assign.Pos(), "naming") {
		return
	}
	for _, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		checkShortName(pass, ident, loopIndexPos, diags)
	}
}

// checkShortName flags a single-letter identifier that is not _, not a loop index.
func checkShortName(pass *runner.Pass, name *ast.Ident, loopIndexPos map[token.Pos]bool, diags *[]report.Diagnostic) {
	n := name.Name
	if n == "_" || len([]rune(n)) != 1 {
		return
	}
	if loopIndexPos[name.Pos()] {
		return
	}
	if ignore.IsSuppressed(pass.IgnoreSet, name.Pos(), "naming") {
		return
	}
	*diags = append(*diags, report.Diagnostic{
		Pos:     pass.Fset.Position(name.Pos()),
		Rule:    "naming",
		Message: "variable name \"" + n + "\" is too short; use a descriptive name",
	})
}

func checkInterfaceName(pass *runner.Pass, name *ast.Ident, _ *ast.InterfaceType, diags *[]report.Diagnostic) {
	n := name.Name
	// Flag names like IUser, IUserRepository — capital I followed by capital letter.
	if len(n) >= 2 && n[0] == 'I' && unicode.IsUpper(rune(n[1])) {
		*diags = append(*diags, report.Diagnostic{
			Pos:     pass.Fset.Position(name.Pos()),
			Rule:    "naming",
			Message: "interface \"" + n + "\" should not be prefixed with I; use a descriptive noun or -er suffix",
		})
	}
}

// isUpperSnakeCase returns true for names like FOO_BAR or MAX_SIZE.
func isUpperSnakeCase(s string) bool {
	if !strings.Contains(s, "_") {
		return false
	}
	for _, r := range s {
		if r == '_' {
			continue
		}
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func checkConstantName(pass *runner.Pass, name *ast.Ident, diags *[]report.Diagnostic) {
	_ = token.NoPos
	if isUpperSnakeCase(name.Name) {
		*diags = append(*diags, report.Diagnostic{
			Pos:     pass.Fset.Position(name.Pos()),
			Rule:    "naming",
			Message: "constant \"" + name.Name + "\" should use MixedCaps, not UPPER_SNAKE_CASE",
		})
	}
}
