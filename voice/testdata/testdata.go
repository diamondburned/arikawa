package testdata

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/pkg/errors"
)

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
