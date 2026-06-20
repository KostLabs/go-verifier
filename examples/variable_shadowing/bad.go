package variable_shadowing

// name is shadowed across multiple nested blocks.
func BuildName(prefix string) string {
	name := prefix
	{
		name := name + "-inner" // shadows outer name
		_ = name
	}
	return name
}
