package testdata

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const Nico = "testdata/nico.dca"

// WriteOpus reads the given file containing the Opus frames into the give
// io.Writer.
func WriteOpus(w io.Writer, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", file, err)
	}
	defer f.Close()

	var lenbuf [4]byte
	for {
		_, err := io.ReadFull(f, lenbuf[:])
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Read the integer
		framelen := int64(binary.LittleEndian.Uint32(lenbuf[:]))

		// Copy the frame.
		_, err = io.CopyN(w, f, framelen)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to write: %w", err)
		}
	}
}

// WriterFunc wraps f to be an io.Writer.
type WriterFunc func([]byte) (int, error)

func (w WriterFunc) Write(b []byte) (int, error) {
	return w(b)
}
