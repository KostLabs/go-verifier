package analyzer

import "testing"

func TestInterfaceSegregation(t *testing.T) {
	a := InterfaceSegregation{}

	tests := []struct {
		name      string
		src       string
		wantRules []string
	}{
		{
			name: "interface with too many methods is flagged",
			src: `package p
type BigService interface {
	Create()
	Read()
	Update()
	Delete()
	List()
	Search()
}`,
			wantRules: []string{"interface-segregation"},
		},
		{
			name: "interface at the threshold is not flagged",
			src: `package p
type SmallService interface {
	Create()
	Read()
	Update()
	Delete()
	List()
}`,
			wantRules: nil,
		},
		{
			name: "small interface is not flagged",
			src: `package p
type Reader interface {
	Read() ([]byte, error)
}`,
			wantRules: nil,
		},
		{
			name: "ignore directive suppresses the finding",
			src: `package p
//goverifier:ignore:interface-segregation
type BigService interface {
	Create()
	Read()
	Update()
	Delete()
	List()
	Search()
}`,
			wantRules: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runAnalyzer(t, a, "p", tc.src)
			assertDiags(t, got, tc.wantRules...)
		})
	}
}
