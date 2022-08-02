package vault

import (
	"strings"
	"testing"
)

func TestIsPathValidBucketPrefix(t *testing.T) {
	const P = "IA-DPS-VAULT"
	cases := []struct {
		About        string
		BucketPrefix string
		Path         string
		Result       bool
	}{
		{"cannot be empty", P, "", false},
		{"not just slash", P, "/", false},
		{"invalid prefix suffix combination", P, "/IA-DPS-VAULT-xyz_files.xml", false},
		{"invalid prefix suffix combination", P, "/IA-DPS-VAULT-xyz_meta.xml", false},
		{"invalid prefix suffix combination", P, "/IA-DPS-VAULT-xyz_meta.sqlite", false},
		{"invalid prefix suffix combination", P, "/IA-DPS-VAULT-xyz_reviews.xml", false},
		{"no dot", P, "/./", false},
		{"no dotdot", P, "/./..", false},
		{"no double slash", P, "/a//b", false},
		{"name max", P, "/a/" + strings.Repeat("b", 256) + "/c", false},
		{"path max", P, strings.Repeat("/abc", 1025), false},
		{"invalid byte", P, "ab\x00c", false},
		{"invalid byte", P, "ab\x0ac", false},
		{"invalid byte", P, "ab\x0dc", false},
		{"illegal xml", P, "ab\x11c", false},
	}
	for _, c := range cases {
		result := IsValidPathPrefix(c.Path, c.BucketPrefix)
		if result != c.Result {
			t.Errorf("[%v] got %v, want %v", c.Path, result, c.Result)
		}
	}
}
