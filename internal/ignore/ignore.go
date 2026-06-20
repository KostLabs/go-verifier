// Package ignore parses //goverifier:ignore directives from Go source files.
// A directive applies to the next AST node following the comment.
// Format: //goverifier:ignore:<rule1>,<rule2>  or  //goverifier:ignore  (suppresses all)
package ignore

import (
	"go/ast"
	"go/token"
	"strings"
)

const prefix = "//goverifier:ignore"

// Set maps token positions to the set of suppressed rule names for that node.
// The special value "*" means all rules are suppressed.
type Set map[token.Pos]map[string]struct{}

// Parse builds an ignore Set from the comments in a file.
// It associates each directive with the position of the next declaration or
// statement that follows it in the file.
func Parse(fset *token.FileSet, file *ast.File) Set {
	set := make(Set)

	// Collect all directive comments with their end positions.
	type entry struct {
		end   token.Pos
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
			rest := strings.TrimPrefix(text, prefix)
			if rest == "" {
				// //goverifier:ignore — suppress all
				rules["*"] = struct{}{}
			} else if after, ok := strings.CutPrefix(rest, ":"); ok {
				for _, r := range strings.Split(after, ",") {
					r = strings.TrimSpace(r)
					if r != "" {
						rules[r] = struct{}{}
					}
				}
			}
			if len(rules) > 0 {
				directives = append(directives, entry{end: c.End(), rules: rules})
			}
		}
	}

	if len(directives) == 0 {
		return set
	}

	// Collect positions of all top-level declarations and their inner nodes.
	var nodePositions []token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		nodePositions = append(nodePositions, n.Pos())
		return true
	})

	// For each directive, find the first node that starts after the comment ends.
	for _, d := range directives {
		for _, pos := range nodePositions {
			if pos > d.end {
				set[pos] = d.rules
				break
			}
		}
	}

	return set
}

// IsSuppressed reports whether rule is suppressed at the given position.
func IsSuppressed(set Set, pos token.Pos, rule string) bool {
	rules, ok := set[pos]
	if !ok {
		return false
	}
	if _, all := rules["*"]; all {
		return true
	}
	_, found := rules[rule]
	return found
}
