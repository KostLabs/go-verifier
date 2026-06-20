// Package runner loads Go packages and runs goverifier analyzers over them.
package runner

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"

	"goverifier/internal/ignore"
	"goverifier/internal/report"
)

// Analyzer is the interface every rule checker implements.
type Analyzer interface {
	// Name is the rule identifier used in //goverifier:ignore directives.
	Name() string
	// Run checks a single file and appends diagnostics to the provided slice.
	// ignoreSet lets analyzers skip nodes suppressed by directives.
	Run(pass *Pass) []report.Diagnostic
}

// Pass carries the per-file context passed to each Analyzer.
type Pass struct {
	Fset      *token.FileSet
	File      *ast.File
	TypesInfo *types.Info
	Pkg       *types.Package
	IgnoreSet ignore.Set
}

// Config controls which analyzers are active.
type Config struct {
	Analyzers []Analyzer
	Enabled   map[string]bool // nil means all enabled
	Disabled  map[string]bool
}

func (c *Config) active(name string) bool {
	if len(c.Disabled) > 0 && c.Disabled[name] {
		return false
	}
	if len(c.Enabled) > 0 {
		return c.Enabled[name]
	}
	return true
}

// Run loads the given patterns and runs all active analyzers, returning all diagnostics.
func Run(patterns []string, cfg Config) ([]report.Diagnostic, error) {
	pkgCfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Tests: false,
	}

	pkgs, err := packages.Load(pkgCfg, patterns...)
	if err != nil {
		return nil, err
	}

	var results []report.Diagnostic

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			// Surface package load errors as diagnostics so the caller sees them.
			for _, e := range pkg.Errors {
				results = append(results, report.Diagnostic{
					Pos:     token.Position{Filename: e.Pos},
					Rule:    "load-error",
					Message: e.Msg,
				})
			}
			continue
		}

		for _, file := range pkg.Syntax {
			ignoreSet := ignore.Parse(pkg.Fset, file)
			pass := &Pass{
				Fset:      pkg.Fset,
				File:      file,
				TypesInfo: pkg.TypesInfo,
				Pkg:       pkg.Types,
				IgnoreSet: ignoreSet,
			}

			for _, a := range cfg.Analyzers {
				if !cfg.active(a.Name()) {
					continue
				}
				diags := a.Run(pass)
				results = append(results, diags...)
			}
		}
	}

	return results, nil
}
