// Package iotemp implements helpers around io and temporary files.
package iotemp

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
)

// DummyReader reads a fixed number of dummy bytes, e.g. dots; placeholder.
type DummyReader struct {
	N int64 // max
	C byte  // char to use
	i int64 // current
}

// Read reports reads, but does nothing.
func (r *DummyReader) Read(p []byte) (n int, err error) {
	if r.N == 0 {
		return 0, io.EOF
	}
	if r.N < 0 {
		return 0, fmt.Errorf("N must be positive")
	}
	for i := range p {
		p[i] = r.C
	}
	l := int64(len(p))
	if r.i+l > r.N {
		// https://i.imgur.com/2Zm3WHd.png
		s := fmt.Sprintf("%d", r.N)
		ls := int64(len(s))
		if r.N-r.i-2-ls > 0 {
			for i, c := range s {
				p[r.N-r.i-2-(ls-int64(i))] = byte(c)
			}
		}
		p[r.N-r.i-1] = 0x0a
		return int(r.N - r.i), io.EOF
	}
	r.i += l
	return len(p), nil
}

// TempFileFromReader spools a reader into temporary file and returns its name.
func TempFileFromReader(r io.Reader) (string, error) {
	tf, err := ioutil.TempFile("", "rclone-vault-transit-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tf, r); err != nil {
		return "", err
	}
	if err := tf.Close(); err != nil {
		return "", err
	}
	return tf.Name(), nil
}

// MultipartFieldWriter retains any error that may have occured.
type MultipartFieldWriter struct {
	W   *multipart.Writer
	err error
}

func (w *MultipartFieldWriter) WriteField(fieldname, value string) {
	if w.err != nil {
		return
	}
	w.err = w.W.WriteField(fieldname, value)
}

func (w *MultipartFieldWriter) Err() error {
	return w.err
}
