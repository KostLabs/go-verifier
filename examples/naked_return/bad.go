package naked_return

import "errors"

// Named return values with bare return — hides what the function actually returns.
func Divide(a, b float64) (result float64, err error) {
	if b == 0 {
		err = errors.New("division by zero")
		return // naked return: obscures that result=0, err=err
	}
	result = a / b
	return // naked return: obscures that result=a/b, err=nil
}

// Multiple named returns, all naked.
func ParsePair(s string) (first, second string, err error) {
	if s == "" {
		err = errors.New("empty input")
		return
	}
	first = s[:len(s)/2]
	second = s[len(s)/2:]
	return
}
