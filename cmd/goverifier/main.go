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

// ruleFlag is a flag.Value that accumulates rule names from repeated -flag or
// comma-separated values: -disable foo -disable bar,baz all work.
type ruleFlag map[string]bool

func (rf ruleFlag) String() string { return "" }
func (rf ruleFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			rf[part] = true
		}
	}
	return nil
}

func main() {
	enable := make(ruleFlag)
	disable := make(ruleFlag)
	format := flag.String("format", "text", "output format: text or json")
	flag.Var(enable, "enable", "rule to enable; may be repeated or comma-separated (default: all)")
	flag.Var(disable, "disable", "rule to disable; may be repeated or comma-separated")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := runner.Config{
		Analyzers: analyzer.All(),
	}
	if len(enable) > 0 {
		cfg.Enabled = enable
	}
	if len(disable) > 0 {
		cfg.Disabled = disable
	}

	diags, err := runner.Run(patterns, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var outputFormat report.Format
	switch *format {
	case "json":
		outputFormat = report.FormatJSON
	default:
		outputFormat = report.FormatText
	}

	if writeErr := report.Write(os.Stdout, diags, outputFormat); writeErr != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", writeErr)
		os.Exit(1)
	}

	if len(diags) > 0 {
		os.Exit(1)
	}
}
