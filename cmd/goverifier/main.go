package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/KostLabs/go-verifier/internal/analyzer"
	"github.com/KostLabs/go-verifier/internal/report"
	"github.com/KostLabs/go-verifier/internal/runner"
)

const usage = `go-verifier: enforce KLabs Go practices

Usage:
  goverifier [flags] [packages...]

Flags:
  -enable  comma-separated list of rules to run (default: all)
  -disable comma-separated list of rules to skip
  -format  output format: text (default) or json

Examples:
  goverifier ./...
  goverifier -enable context-propagation,error-handling ./pkg/...
  goverifier -disable naming ./...
  goverifier -format json ./...

Ignore directives (place on the line before the node):
  //goverifier:ignore                        suppress all rules on next node
  //goverifier:ignore:context-propagation    suppress one rule
  //goverifier:ignore:naming,defer-in-loop   suppress multiple rules

Available rules:
  context-propagation   context.Context must be first parameter
  error-handling        no panic; wrap errors with %%w; error last in returns
  naming                no I-prefixed interfaces; no UPPER_SNAKE_CASE consts; no generic package names
  defer-in-loop         no defer inside for/range loops
  naked-return          no bare return in named-result functions
  else-usage            no else after always-exiting if block
  logging               no fmt.Print*/log.Print* in production code
  any-type              no any/interface{} outside marshalling contexts
  variable-shadowing    no inner-scope variable shadowing outer variable
  interface-definition  interfaces should not be defined alongside their implementation
`

func main() {
	enable := flag.String("enable", "", "comma-separated rules to enable (default: all)")
	disable := flag.String("disable", "", "comma-separated rules to disable")
	format := flag.String("format", "text", "output format: text or json")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := runner.Config{
		Analyzers: analyzer.All(),
	}

	if *enable != "" {
		cfg.Enabled = splitSet(*enable)
	}
	if *disable != "" {
		cfg.Disabled = splitSet(*disable)
	}

	diags, err := runner.Run(patterns, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var fmt_ report.Format
	switch *format {
	case "json":
		fmt_ = report.FormatJSON
	default:
		fmt_ = report.FormatText
	}

	writeErr := report.Write(os.Stdout, diags, fmt_)
	if writeErr != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", writeErr)
		os.Exit(1)
	}

	if len(diags) > 0 {
		os.Exit(1)
	}
}

func splitSet(s string) map[string]bool {
	m := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			m[part] = true
		}
	}
	return m
}
