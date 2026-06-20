package variable_shadowing

import "errors"

// err is declared in the outer scope then re-declared (shadowed) in a nested scope.
func LoadAndProcess() error {
	err := doLoad()
	if err != nil {
		return err
	}

	if true {
		// This err shadows the outer err — changes here don't affect the outer one.
		result, err := doProcess()
		if err != nil {
			return err
		}
		_ = result
	}

	return nil
}

// name is shadowed across multiple nested blocks.
func BuildName(prefix string) string {
	name := prefix
	{
		name := name + "-inner" // shadows outer name
		_ = name
	}
	return name
}

func doLoad() error          { return nil }
func doProcess() (int, error) { return 0, errors.New("process error") }
