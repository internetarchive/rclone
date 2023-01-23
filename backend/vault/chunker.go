package vault

import (
	"io"
	"math"
	"os"
)

// Chunker allows to read file in chunks of fixed sizes.
type Chunker struct {
	chunkSize int64 // in bytes
	fileSize  int64
	numChunks int64
	f         *os.File
}

// NewChunker sets up a new chunker. Caller will need to close this to close
// the associated file. TODO: we can get rid of a filename here and use an
// io.ReaderAt.
func NewChunker(filename string, chunkSize int64) (*Chunker, error) {
	if chunkSize < 1 {
		return nil, ErrInvalidChunkSize
	}
	var (
		f         *os.File
		fi        os.FileInfo
		err       error
		numChunks int64
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

// ChunkReader returns the reader over a section of the file. Counting starts at zero.
func (c *Chunker) ChunkReader(i int64) io.Reader {
	offset := i * c.chunkSize
	return io.NewSectionReader(c.f, offset, c.chunkSize)
}

// Close closes the wrapped file.
func (c *Chunker) Close() error {
	return c.f.Close()
}
