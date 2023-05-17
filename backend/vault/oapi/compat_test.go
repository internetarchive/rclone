package oapi

import "testing"

func TestSafeDereference(t *testing.T) {
	var (
		s  = "hello"
		i  = 123
		b  = true
		ps *string
		pi *int
	)
	var cases = []struct {
		v      any
		result any
	}{
		{nil, nil},
		{0, 0},
		{"", ""},
		{"hello", "hello"},
		{&s, "hello"},
		{&i, 123},
		{&b, true},
		{ps, nil},
		{pi, nil},
	}
	for _, c := range cases {
		result := safeDereference(c.v)
		if result != c.result {
			t.Fatalf("got %v, want %v", result, c.result)
		}
	}
}
