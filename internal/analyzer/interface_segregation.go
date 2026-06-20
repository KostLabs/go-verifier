package analyzer

import (
	"fmt"
	"go/ast"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// InterfaceSegregation flags interfaces that declare too many methods.
// Fat interfaces force implementors to provide methods they may not need,
// violating the Interface Segregation Principle.
//
// Threshold: more than 5 methods triggers a finding.
type InterfaceSegregation struct{}

func (InterfaceSegregation) Name() string { return "interface-segregation" }

const interfaceMethodThreshold = 5

func (InterfaceSegregation) Run(pass *runner.Pass) []report.Diagnostic {
	var diags []report.Diagnostic

	ast.Inspect(pass.File, func(n ast.Node) bool {
		gd, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		// Directive before the type block maps to the GenDecl position.
		if ignore.IsSuppressed(pass.IgnoreSet, gd.Pos(), "interface-segregation") {
			return false
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			if ignore.IsSuppressed(pass.IgnoreSet, ts.Pos(), "interface-segregation") {
				continue
			}

			count := 0
			for _, method := range iface.Methods.List {
				if len(method.Names) > 0 {
					count += len(method.Names)
				}
			}

			if count > interfaceMethodThreshold {
				diags = append(diags, report.Diagnostic{
					Pos:  pass.Fset.Position(ts.Pos()),
					Rule: "interface-segregation",
					Message: fmt.Sprintf(
						"interface %q has %d methods; consider splitting it into smaller focused interfaces (max %d)",
						ts.Name.Name, count, interfaceMethodThreshold,
					),
				})
			}
		}

		return true
	})

	return diags
}
