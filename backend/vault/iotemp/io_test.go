package iotemp

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestDummyReader(t *testing.T) {
	var cases = []struct {
		about    string
		r        *DummyReader
		expected []byte
	}{
		{
			"output is N bytes long",
			&DummyReader{N: 4, C: '.'},
			[]byte{'.', '4', '.', '\n'},
		},
		{
			"3 byte output",
			&DummyReader{N: 3, C: '.'},
			[]byte{'.', '.', '\n'},
		},
		{
			"2 byte output",
			&DummyReader{N: 2, C: '.'},
			[]byte{'.', '\n'},
		},
		{
			"1 byte output",
			&DummyReader{N: 1, C: '.'},
			[]byte{'\n'},
		},
		{
			"0 byte output",
			&DummyReader{N: 0, C: '.'},
			[]byte{},
		},
	}
	for _, c := range cases {
		b, err := ioutil.ReadAll(c.r)
		if err != nil {
			t.Fatalf("read failed [%s]: %v", c.about, err)
		}
		if string(b) != string(c.expected) {
			t.Fatalf("[%s]: got %v, want %v", c.about, b, c.expected)
		}
	}
}

func TestTempFileFromReader(t *testing.T) {
	const s = "hello"
	r := strings.NewReader(s)
	filename, err := TempFileFromReader(r)
	if err != nil {
		t.Fatalf("tempfile from reader failed: %v", err)
	}
	if !strings.HasPrefix(filename, os.TempDir()) {
		t.Fatalf("tempfile should be under tempdir")
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("read from tempfile failed: %v", err)
	}
	if string(b) != s {
		t.Fatalf("tempfile reader mismatch, got %v, want %v", string(b), s)
	}
	_ = os.Remove(filename)
}
