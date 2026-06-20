package analyzer

import (
	"testing"
)

func TestContextPropagation(t *testing.T) {
	a := ContextPropagation{}

	tests := []struct {
		name      string
		given     string // source code description
		src       string
		wantRules []string // expected rule names; empty means no findings
	}{
		{
			name:  "function calling context-aware API without ctx parameter",
			given: "a function that calls db.QueryRowContext but does not accept context.Context",
			src: `package p
import (
	"context"
	"database/sql"
)
var db *sql.DB
func GetUser(id int) (*sql.Row, error) {
	row := db.QueryRowContext(context.Background(), "SELECT 1")
	return row, nil
}`,
			wantRules: []string{"context-propagation"},
		},
		{
			name:  "function passing ctx variable without accepting it as parameter",
			given: "a function that creates a local ctx and passes it to another function",
			src: `package p
import (
	"context"
	"database/sql"
)
var db *sql.DB
func DeleteUser(id int) error {
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}`,
			wantRules: []string{"context-propagation"},
		},
		{
			name:  "function already accepting context.Context as first parameter",
			given: "a function with context.Context as its first parameter",
			src: `package p
import (
	"context"
	"database/sql"
)
var db *sql.DB
func GetUser(ctx context.Context, id int) (*sql.Row, error) {
	row := db.QueryRowContext(ctx, "SELECT 1")
	return row, nil
}`,
			wantRules: nil,
		},
		{
			name:  "function with no I/O needs no context",
			given: "a pure function that does no I/O",
			src: `package p
func Add(a, b int) int {
	return a + b
}`,
			wantRules: nil,
		},
		{
			name:  "ignore directive suppresses the finding",
			given: "a context-propagation violation suppressed by //goverifier:ignore",
			src: `package p
import (
	"context"
	"database/sql"
)
var db *sql.DB
//goverifier:ignore:context-propagation
func GetUser(id int) (*sql.Row, error) {
	row := db.QueryRowContext(context.Background(), "SELECT 1")
	return row, nil
}`,
			wantRules: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// When: the analyzer runs on the given source
			got := runAnalyzer(t, a, "p", tc.src)

			// Then: diagnostics match expectations
			assertDiags(t, got, tc.wantRules...)
		})
	}
}
