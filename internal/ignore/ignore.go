// Package ignore parses //goverifier:ignore directives from Go source files.
//
// A directive suppresses findings on the node it annotates. Two placements are
// supported:
//
//   - Preceding line:  the directive appears on the line immediately before the
//     node (or before the GenDecl / FuncDecl that owns it).
//   - Inline (same line): the directive appears on the same source line as the
//     node's opening position.
//
// Format: //goverifier:ignore:<rule1>,<rule2> [optional reason text]
// or:     //goverifier:ignore  (suppresses all rules)
package ignore

import (
	"go/ast"
	"go/token"
	"strings"
)

const prefix = "//goverifier:ignore"

// Set holds parsed directives indexed two ways for fast lookup:
//   - byPos: keyed by the exact token.Pos of the annotated node (preceding-line placement)
//   - byLine: keyed by source line number (inline placement)
type Set struct {
	byPos  map[token.Pos]map[string]struct{}
	byLine map[int]map[string]struct{}
	fset   *token.FileSet
}

// Parse builds a Set from the comments in a file.
func Parse(fset *token.FileSet, file *ast.File) Set {
	set := Set{
		byPos:  make(map[token.Pos]map[string]struct{}),
		byLine: make(map[int]map[string]struct{}),
		fset:   fset,
	}

	type entry struct {
		line  int       // source line the comment is on
		end   token.Pos // end position of the comment token
		rules map[string]struct{}
	}
	var directives []entry

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			text := strings.TrimSpace(c.Text)
			if !strings.HasPrefix(text, prefix) {
				continue
			}
			rules := make(map[string]struct{})
			if rest := strings.TrimPrefix(text, prefix); rest == "" {
				rules["*"] = struct{}{}
			} else if after, ok := strings.CutPrefix(rest, ":"); ok {
				// Everything up to the first whitespace is the rule list; the rest
				// is an optional freetext reason.
				if parts := strings.FieldsFunc(after, func(r rune) bool { return r == ' ' || r == '\t' }); len(parts) > 0 {
					for _, r := range strings.Split(parts[0], ",") {
						r = strings.TrimSpace(r)
						if r != "" {
							rules[r] = struct{}{}
						}
					}
				}
			}
			if len(rules) > 0 {
				line := fset.Position(c.Pos()).Line
				directives = append(directives, entry{line: line, end: c.End(), rules: rules})
			}
		}
	}

	if len(directives) == 0 {
		return set
	}

	// Index inline directives by their source line so IsSuppressed can match
	// nodes whose Pos() falls on the same line as the comment.
	for _, d := range directives {
		set.byLine[d.line] = d.rules
	}

	// Index preceding-line directives: associate each directive with the first
	// node that begins strictly after the comment ends.
	var nodePositions []token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		nodePositions = append(nodePositions, n.Pos())
		return true
	})

	for _, d := range directives {
		for _, pos := range nodePositions {
			if pos > d.end {
				set.byPos[pos] = d.rules
				break
			}
		}
	}

	return set
}

// IsSuppressed reports whether rule is suppressed at the given position.
// It matches both preceding-line directives (exact pos lookup) and inline
// directives (same source line as pos).
func IsSuppressed(set Set, pos token.Pos, rule string) bool {
	if rules, ok := set.byPos[pos]; ok {
		if _, all := rules["*"]; all {
			return true
		}
		if _, found := rules[rule]; found {
			return true
		}
	}
	if set.fset != nil {
		line := set.fset.Position(pos).Line
		if rules, ok := set.byLine[line]; ok {
			if _, all := rules["*"]; all {
				return true
			}
			if _, found := rules[rule]; found {
				return true
			}
		}
	}
	return false
}
