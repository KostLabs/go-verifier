package analyzer

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
	"goverifier/internal/runner"
)

// runAnalyzer parses src as a Go source file in the given package name,
// type-checks it, and runs a against it. Returns the diagnostics produced.
func runAnalyzer(t *testing.T, a runner.Analyzer, pkgName, src string) []report.Diagnostic {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "input.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Override the package name in the AST so callers can set it freely.
	f.Name.Name = pkgName

	cfg := &types.Config{
		Importer: importer.Default(),
		Error:    func(e error) {}, // suppress type errors in intentionally bad code
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	pkg, _ := cfg.Check(pkgName, fset, []*ast.File{f}, info)

	pass := &runner.Pass{
		Fset:      fset,
		File:      f,
		TypesInfo: info,
		Pkg:       pkg,
		IgnoreSet: ignore.Parse(fset, f),
	}

	return a.Run(pass)
}

// assertDiags checks that exactly the expected rule names appear (order-independent).
func assertDiags(t *testing.T, got []report.Diagnostic, wantRules ...string) {
	t.Helper()
	if len(got) != len(wantRules) {
		gotRules := make([]string, len(got))
		for i, d := range got {
			gotRules[i] = d.Rule + "@" + d.Pos.String() + ": " + d.Message
		}
		t.Errorf("got %d diagnostics, want %d\ngot:  %v\nwant: %v", len(got), len(wantRules), gotRules, wantRules)
		return
	}
	counts := make(map[string]int)
	for _, r := range wantRules {
		counts[r]++
	}
	for _, d := range got {
		counts[d.Rule]--
	}
	for rule, delta := range counts {
		if delta != 0 {
			t.Errorf("rule %q: count off by %d (negative = too many, positive = missing)", rule, delta)
		}
	}
}

// assertNoDiags asserts the analyzer produced no findings.
func assertNoDiags(t *testing.T, got []report.Diagnostic) {
	t.Helper()
	if len(got) != 0 {
		for _, d := range got {
			t.Errorf("unexpected diagnostic [%s] %s at %s", d.Rule, d.Message, d.Pos)
		}
	}
}

// runAnalyzerInTestFile parses src using a _test.go filename so analyzers
// that skip test files (e.g. logging) behave accordingly.
func runAnalyzerInTestFile(t *testing.T, a runner.Analyzer, src string) []report.Diagnostic {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "input_test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	cfg := &types.Config{
		Importer: importer.Default(),
		Error:    func(e error) {},
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	pkg, _ := cfg.Check("p", fset, []*ast.File{f}, info)

	pass := &runner.Pass{
		Fset:      fset,
		File:      f,
		TypesInfo: info,
		Pkg:       pkg,
		IgnoreSet: ignore.Parse(fset, f),
	}

	return a.Run(pass)
}
