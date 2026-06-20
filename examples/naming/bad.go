package utils

// Interface prefixed with I — non-idiomatic in Go.
type IUserRepository interface {
	Find(id int) error
	Save(id int) error
}

// Interface prefixed with I (single-method).
type IReader interface {
	Read(p []byte) (int, error)
}

// Constants using UPPER_SNAKE_CASE instead of MixedCaps.
const (
	MAX_RETRIES     = 3
	DEFAULT_TIMEOUT = 30
	MIN_PASSWORD_LENGTH = 8
)
