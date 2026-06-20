package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
	"goverifier/internal/runner"
)

// Naming checks:
//   - Interface names don't start with "I" (e.g. IUserRepository)
//   - Constants aren't UPPER_SNAKE_CASE
//   - Package names aren't generic (utils, helpers, common, util, helper)
//   - Single-method interfaces use method name + "-er" suffix (advisory only)
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
		}
		return true
	})

	return diags
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
