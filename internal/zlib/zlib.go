// Package zlib provides abstractions on top of compress/zlib to work with
// Discord's method of compressing websocket packets.
package zlib

import (
	"bytes"
	"log"

	"github.com/pkg/errors"
)

var Suffix = [4]byte{'\x00', '\x00', '\xff', '\xff'}

var ErrPartial = errors.New("only partial payload in buffer")

type Inflator struct {
	zlib Reader
	wbuf bytes.Buffer // write buffer for writing compressed bytes
	rbuf bytes.Buffer // read buffer for writing uncompressed bytes
}

func NewInflator() *Inflator {
	return &Inflator{
		wbuf: bytes.Buffer{},
		rbuf: bytes.Buffer{},
	}
}

func (i *Inflator) Write(p []byte) (n int, err error) {
	log.Println(p)
	// Write to buffer normally.
	return i.wbuf.Write(p)
}

// CanFlush returns if Flush() should be called.
func (i *Inflator) CanFlush() bool {
	if i.wbuf.Len() < 4 {
		return false
	}
	p := i.wbuf.Bytes()
	return bytes.Equal(p[len(p)-4:], Suffix[:])
}

func (i *Inflator) Flush() ([]byte, error) {
	// Check if close frames are there:
	// if !i.CanFlush() {
	// 	return nil, ErrPartial
	// }

	// log.Println(i.wbuf.Bytes())

	// We should reset the write buffer after flushing.
	// defer i.wbuf.Reset()

	// We can reset the read buffer while returning its byte slice. This works
	// as long as we copy the byte slice before resetting.
	defer i.rbuf.Reset()

	// Guarantee there's a zlib writer. Since Discord streams zlib, we have to
	// reuse the same Reader. Only the first packet has the zlib header.
	if i.zlib == nil {
		r, err := zlibStreamer(&i.wbuf)
		if err != nil {
			return nil, errors.Wrap(err, "failed to make a FLATE reader")
		}
		// safe assertion
		i.zlib = r
		// } else {
		// 	// Reset the FLATE reader for future use:
		// 	if err := i.zlib.Reset(&i.wbuf, nil); err != nil {
		// 		return nil, errors.Wrap(err, "failed to reset zlib reader")
		// 	}
	}

	// We can ignore zlib.Read's error, as zlib.Close would return them.
	_, err := i.rbuf.ReadFrom(i.zlib)

	// ErrUnexpectedEOF happens because zlib tries to find the last 4 bytes
	// to verify checksum. Discord doesn't send this.
	if err != nil {
		// Unexpected error, try and close.
		return nil, errors.Wrap(err, "failed to read from FLATE reader")
	}

	// 	if err := i.zlib.Close(); err != nil && err != io.ErrUnexpectedEOF {
	// 		// Try and close anyway.
	// 		return nil, errors.Wrap(err, "failed to read from zlib reader")
	// 	}

	// Copy the bytes.
	return bytecopy(i.rbuf.Bytes()), nil
}

// func (d *Deflator) TryFlush() ([]byte, error) {
// 	// Check if the buffer ends with the zlib close suffix.
// 	if d.wbuf.Len() < 4 {
// 		return nil, nil
// 	}
// 	if p := d.wbuf.Bytes(); !bytes.Equal(p[len(p)-4:], Suffix[:]) {
// 		return nil, nil
// 	}

// 	// Guarantee there's a zlib writer. Since Discord streams zlib, we have to
// 	// reuse the same Reader. Only the first packet has the zlib header.
// 	if d.zlib == nil {
// 		r, err := zlib.NewReader(&d.wbuf)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "failed to make a zlib reader")
// 		}
// 		// safe assertion
// 		d.zlib = r
// 	}

// 	// We can reset the read buffer while returning its byte slice. This works
// 	// as long as we copy the byte slice before resetting.
// 	defer d.rbuf.Reset()

// 	defer d.wbuf.Reset()

// 	// We can ignore zlib.Read's error, as zlib.Close would return them.
// 	_, err := d.rbuf.ReadFrom(d.zlib)
// 	log.Println("Read:", err, d.rbuf.String())

// 	// ErrUnexpectedEOF happens because zlib tries to find the last 4 bytes
// 	// to verify checksum. Discord doesn't send this.
// 	// if err != nil && err != io.ErrUnexpectedEOF {
// 	// 	// Unexpected error, try and close.
// 	// 	return nil, errors.Wrap(err, "failed to read from zlib reader")
// 	// }

// 	if err := d.zlib.Close(); err != nil && err != io.ErrUnexpectedEOF {
// 		// Try and close anyway.
// 		return nil, errors.Wrap(err, "failed to read from zlib reader")
// 	}

// 	// Copy the bytes.
// 	return bytecopy(d.rbuf.Bytes()), nil
// }

func bytecopy(p []byte) []byte {
	cpy := make([]byte, len(p))
	copy(cpy, p)
	return cpy
}
