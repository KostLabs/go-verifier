package defer_in_loop

import "os"

// defer inside a for loop — defers stack until function returns,
// not until each iteration ends.
func ProcessFiles(paths []string) error {
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	return nil
}

// defer inside a classic for loop.
func CountDown(n int) {
	for i := n; i > 0; i-- {
		defer func(v int) {
			_ = v
		}(i)
	}
}
