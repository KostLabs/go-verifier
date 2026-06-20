# go-verifier

A static analysis tool that enforces [KLabs Go coding practices](https://practices.kostlabs.org).

## Installation

```bash
go install github.com/KostLabs/go-verifier/cmd/goverifier@latest
```

Or build from source:

```bash
git clone https://github.com/KostLinux/go-verifier.git
cd go-verifier
go build -o goverifier ./cmd/goverifier
```

## Usage

```
goverifier [flags] [packages...]
```

If no packages are specified, `./...` is used by default.

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-enable` | Comma-separated list of rules to run | all rules |
| `-disable` | Comma-separated list of rules to skip | none |
| `-format` | Output format: `text` or `json` | `text` |

### Examples

Run all rules on the entire module:

```bash
goverifier ./...
```

Run only specific rules:

```bash
goverifier -enable context-propagation,error-handling ./pkg/...
```

Disable a specific rule:

```bash
goverifier -disable naming ./...
```

Output findings as JSON:

```bash
goverifier -format json ./...
```

## Rules

| Rule | Description |
|------|-------------|
| `context-propagation` | `context.Context` must be the first parameter of any function that uses it |
| `error-handling` | No `panic`; wrap errors with `%w`; `error` must be the last return value |
| `naming` | No `I`-prefixed interfaces; no `UPPER_SNAKE_CASE` constants; no generic package names |
| `defer-in-loop` | No `defer` inside `for`/`range` loops |
| `naked-return` | No bare `return` in functions with named result parameters |
| `else-usage` | No `else` after an `if` block that always exits (return/panic/continue) |
| `logging` | No `fmt.Print*` or `log.Print*` in production code |
| `any-type` | No `any`/`interface{}` outside marshalling contexts |
| `variable-shadowing` | No inner-scope variable that shadows an outer-scope variable |
| `interface-definition` | Interfaces should not be defined in the same package as their implementation |

## Ignore Directives

Place a directive on the line immediately before the node you want to suppress:

```go
// Suppress all rules on the next node
//goverifier:ignore
func MyFunc() { ... }

// Suppress a single rule
//goverifier:ignore:context-propagation
func GetUser(id int) (*sql.Row, error) { ... }

// Suppress multiple rules
//goverifier:ignore:naming,defer-in-loop
func legacyHandler() { ... }
```

## Output Formats

**Text** (default):

```
./pkg/user/service.go:42:5: [context-propagation] function GetUser uses context-aware API but does not accept context.Context as first parameter
```

**JSON** (`-format json`):

```json
[
  {
    "file": "./pkg/user/service.go",
    "line": 42,
    "column": 5,
    "rule": "context-propagation",
    "message": "function GetUser uses context-aware API but does not accept context.Context as first parameter"
  }
]
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No violations found |
| `1` | One or more violations found, or a runtime error occurred |

## CI Integration

### GitHub Actions

```yaml
- name: Run go-verifier
  run: goverifier ./...
```
