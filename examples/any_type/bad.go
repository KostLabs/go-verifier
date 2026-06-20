package any_type

// Struct field typed as any — loses type safety.
type Config struct {
	Value   any
	Options interface{}
	Extra   map[string]any
}

// Function parameter typed as any.
func Process(input any) any {
	return input
}

// Variable declared as any.
func Run() {
	var result any = "hello"
	_ = result
}
