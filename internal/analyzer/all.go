// Package analyzer contains all goverifier rule implementations.
package analyzer

import "github.com/KostLabs/go-verifier/internal/runner"

// All returns every built-in analyzer.
func All() []runner.Analyzer {
	return []runner.Analyzer{
		ContextPropagation{},
		ErrorHandling{},
		Naming{},
		DeferInLoop{},
		NakedReturn{},
		ElseUsage{},
		Logging{},
		AnyType{},
		VariableShadowing{},
		InterfaceDefinition{},
	}
}
