package vault

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestChunker(t *testing.T) {

	var cases = []struct {
		data           string
		numChunks      int64
		err            error
		expectedChunks []string
	}{
		{"", 0, ErrInvalidChunkSize, []string{}},
		{"", 1, nil, []string{}},
		{"", 2, nil, []string{}},
		{"a", 2, nil, []string{"a"}},
		{"abc", 2, nil, []string{"ab", "c"}},
		{"abcd", 2, nil, []string{"ab", "cd"}},
		{"abcd", 1, nil, []string{"a", "b", "c", "d"}},
	}
	for _, c := range cases {
		f, err := os.CreateTemp(t.TempDir(), "vault-test-chunker*")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err = os.WriteFile(f.Name(), []byte(c.data), 0644); err != nil {
			t.Fatal(err)
		}
		ch, err := NewChunker(f.Name(), c.numChunks)
		if err != nil && err == c.err {
			continue
		}
		if got := ch.NumChunks(); got != int64(len(c.expectedChunks)) {
			t.Fatalf("chunks: got %v, want %v", got, len(c.expectedChunks))
		}
		for i, ec := range c.expectedChunks {
			var buf bytes.Buffer
			chr := ch.ChunkReader(int64(i))
			if _, err = io.Copy(&buf, chr); err != nil {
				t.Fatalf("copy failed: %v", err)
			}
			if buf.String() != ec {
				t.Fatalf("got %v, want %v", buf.String(), ec)
			}
		}
	}
}
