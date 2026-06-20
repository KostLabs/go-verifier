package else_usage

import "errors"

// else after an if that always returns — else is redundant.
func Validate(name string) error {
	if name == "" {
		return errors.New("name is required")
	} else {
		// This else block is unnecessary; the code above always returns.
		if len(name) > 100 {
			return errors.New("name too long")
		} else {
			return nil
		}
	}
}

// else after panic — also an unconditional exit.
func MustBePositive(n int) int {
	if n <= 0 {
		panic("n must be positive")
	} else {
		return n * 2
	}
}
