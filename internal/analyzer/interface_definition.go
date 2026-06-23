package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/KostLabs/go-verifier/internal/ignore"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

// InterfaceDefinition checks that interfaces are not defined in the same
// package as their concrete implementations. Per the practices, interfaces
// should live on the consumer side (the package that uses them), not the
// producer side.
//
// Heuristic: if a package exports both an interface T and a struct that
// implements all of T's methods, it's likely defining the interface on the
// wrong side.
type InterfaceDefinition struct{}

func (InterfaceDefinition) Name() string { return "interface-definition" }

func (InterfaceDefinition) Run(pass *runner.Pass) []report.Diagnostic {
	if pass.TypesInfo == nil {
		return nil
	}

	var diags []report.Diagnostic

	type ifaceEntry struct {
		name  string
		iface *types.Interface
		node  *ast.TypeSpec
	}

	var ifaces []ifaceEntry
	var structTypes []*types.Named

	ast.Inspect(pass.File, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		obj := pass.TypesInfo.Defs[ts.Name]
		if obj == nil {
			return true
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			return true
		}
		switch underlying := named.Underlying().(type) {
		case *types.Interface:
			if !ignore.IsSuppressed(pass.IgnoreSet, ts.Pos(), "interface-definition") {
				ifaces = append(ifaces, ifaceEntry{
					name:  ts.Name.Name,
					iface: underlying,
					node:  ts,
				})
			}
		case *types.Struct:
			structTypes = append(structTypes, named)
		}
		return true
	})

	// For each interface, check if any struct in the same file implements it.
	for _, ie := range ifaces {
		for _, st := range structTypes {
			if structImplementsInterface(st, ie.iface) {
				diags = append(diags, report.Diagnostic{
					Pos:     pass.Fset.Position(ie.node.Pos()),
					Rule:    "interface-definition",
					Message: "interface \"" + ie.name + "\" is defined in the same package as its implementation; consider moving it to the consumer package",
				})
				break
			}
		}
	}

	return diags
}

// structImplementsInterface reports whether the named struct type (or its
// pointer) satisfies every method in iface.
func structImplementsInterface(named *types.Named, iface *types.Interface) bool {
	if iface.NumMethods() == 0 {
		return false // empty interface matches everything — not interesting
	}
	ptrType := types.NewPointer(named)
	return types.Implements(named, iface) || types.Implements(ptrType, iface)
}
