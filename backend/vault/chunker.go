package vault

import (
	"errors"
	"io"
	"math"
	"os"
)

// ErrInvalidChunkSize signals an invalid chunk size.
var ErrInvalidChunkSize = errors.New("invalid chunck size (must be positive)")

// Chunker allows to read file in chunks of fixed sizes. The idea comes from
// https://github.com/flowjs/flow.js, which uses chunks for resilient file
// uploads. This implementation merely deals with reading fixed length parts of
// a file.
type Chunker struct {
	chunkSize int64 // in bytes
	fileSize  int64 // in bytes
	numChunks int64
	f         *os.File
}

// NewChunker sets up a new chunker. Caller will need to close this to close
// the associated file. TODO: we can get rid of a filename here and use an
// io.ReaderAt and some approach similar to io.Scanner.
func NewChunker(filename string, chunkSize int64) (*Chunker, error) {
	if chunkSize < 1 {
		return nil, ErrInvalidChunkSize
	}
	var (
		f   *os.File
		fi  os.FileInfo
		err error
	)
	if f, err = os.Open(filename); err != nil {
		return nil, err
	}
	if fi, err = f.Stat(); err != nil {
		return nil, err
	}
	return &Chunker{
		f:         f,
		chunkSize: chunkSize,
		fileSize:  fi.Size(),
		numChunks: int64(
			math.Ceil(float64(fi.Size()) / float64(chunkSize)),
		)}, nil
}

// FileSize returns the filesize.
func (c *Chunker) FileSize() int64 {
	return c.fileSize
}

// NumChunks returns the number of chunks this file is splitted to.
func (c *Chunker) NumChunks() int64 {
	return c.numChunks
}

// ChunkReader returns the reader over a section of the file. Counting starts
// at zero.
func (c *Chunker) ChunkReader(i int64) io.Reader {
	offset := i * c.chunkSize
	return io.NewSectionReader(c.f, offset, c.chunkSize)
}

// ChunkSize returns the size of a chunk. Counting starts at zero.
func (c *Chunker) ChunkSize(i int64) int64 {
	if i >= 0 && i < (c.numChunks-1) {
		return c.chunkSize
	}
	return c.fileSize - (i * c.chunkSize)
}

// Close closes the wrapped file.
func (c *Chunker) Close() error {
	return c.f.Close()
}
