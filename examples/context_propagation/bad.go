package context_propagation

import (
	"context"
	"database/sql"
)

var db *sql.DB

// Missing context.Context as first parameter, but calls a context-aware API.
func GetUser(id int) (*sql.Row, error) {
	row := db.QueryRowContext(context.Background(), "SELECT * FROM users WHERE id = ?", id)
	return row, nil
}

// Also missing context, passes ctx variable internally.
func DeleteUser(id int) error {
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}
