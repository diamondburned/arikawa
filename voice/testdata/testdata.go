package testdata

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/pkg/errors"
)

const Nico = "testdata/nico.dca"

// WriteOpus reads the given file containing the Opus frames into the give
// io.Writer.
func WriteOpus(w io.Writer, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return errors.Wrap(err, "failed to open "+file)
	}
	defer f.Close()

	var lenbuf [4]byte
	for {
		_, err := io.ReadFull(f, lenbuf[:])
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Wrap(err, "failed to read "+file)
		}

		// Read the integer
		framelen := int64(binary.LittleEndian.Uint32(lenbuf[:]))

		// Copy the frame.
		_, err = io.CopyN(w, f, framelen)
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "failed to write")
		}
	}
}

// WriterFunc wraps f to be an io.Writer.
type WriterFunc func([]byte) (int, error)

func (w WriterFunc) Write(b []byte) (int, error) {
	return w(b)
}
