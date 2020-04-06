package zlib

import (
	"compress/flate"
	"compress/zlib"
	"io"
)

type Reader interface {
	io.ReadCloser
	zlib.Resetter
}

func zlibStreamer(r flate.Reader) (Reader, error) {
	// verify header
	h := make([]byte, 2)

	if _, err := io.ReadFull(r, h); err != nil {
		return nil, err
	}

	// verify header
	if err := verifyHeader(h); err != nil {
		return nil, err
	}

	return flate.NewReader(r).(Reader), nil
}

// https://golang.org/src/compress/zlib/reader.go#L35
const zlibDeflate = 8

func verifyHeader(scratch []byte) error {
	h := uint(scratch[0])<<8 | uint(scratch[1])
	if (scratch[0]&0x0f != zlibDeflate) || (h%31 != 0) {
		return zlib.ErrHeader
	}
	return nil
}
