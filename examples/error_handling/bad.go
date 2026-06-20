package error_handling

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

// Panic instead of returning an error.
func MustParse(s string) int {
	if s == "" {
		panic("empty string")
	}
	return 42
}

// Uses %v instead of %w — breaks errors.Is/As.
func FindUser(id int) error {
	err := ErrNotFound
	return fmt.Errorf("finding user by id failed: %v", err)
}

// error is not the last return value.
func LoadConfig() (error, string) {
	return nil, "config"
}
