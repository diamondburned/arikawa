package ws

import (
	"context"
	"io"
	"net/http"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/pkg/errors"
)

// Codec holds the codec states for Websocket implementations to share with the
// manager. It is used internally in the Websocket and the Connection
// implementation.
type Codec struct {
	Unmarshalers OpUnmarshalers
	Headers      http.Header
}

// NewCodec creates a new default Codec instance.
func NewCodec(unmarshalers OpUnmarshalers) Codec {
	return Codec{
		Unmarshalers: unmarshalers,
		Headers: http.Header{
			"Accept-Encoding": {"zlib"},
		},
	}
}

type codecOp struct {
	Op
	Data json.Raw `json:"d,omitempty"`
}

const maxSharedBufferSize = 1 << 15 // 32KB

// DecodeBuffer boxes a byte slice to provide a shared and thread-unsafe buffer.
// It is used internally and should only be handled around as an opaque thing.
type DecodeBuffer struct {
	buf []byte
}

// NewDecodeBuffer creates a new preallocated DecodeBuffer.
func NewDecodeBuffer(cap int) DecodeBuffer {
	if cap > maxSharedBufferSize {
		cap = maxSharedBufferSize
	}

	return DecodeBuffer{
		buf: make([]byte, 0, cap),
	}
}

// DecodeInto reads the given reader and decodes it into the Op out channel.
//
// buf is optional.
func (c Codec) DecodeInto(ctx context.Context, r io.Reader, buf *DecodeBuffer, out chan<- Op) error {
	var op codecOp
	op.Data = json.Raw(buf.buf)

	if err := json.DecodeStream(r, &op); err != nil {
		return c.send(ctx, out, newErrOp(err, "cannot read JSON stream"))
	}

	if EnableRawEvents {
		dt := op.Data
		op := op.Op
		op.Data = &RawEvent{
			Raw:          dt,
			OriginalCode: op.Code,
			OriginalType: op.Type,
		}
		c.send(ctx, out, op)
	}

	// buf isn't grown from here out. Set it back right now. If Data hasn't been
	// grown, then this will just set buf back to what it was.
	if cap(op.Data) < maxSharedBufferSize {
		buf.buf = op.Data[:0]
	}

	fn := c.Unmarshalers.Lookup(op.Code, op.Type)
	if fn == nil {
		err := UnknownEventError{
			Op:   op.Code,
			Type: op.Type,
		}
		return c.send(ctx, out, newErrOp(err, ""))
	}

	op.Op.Data = fn()
	if err := op.Data.UnmarshalTo(op.Op.Data); err != nil {
		return c.send(ctx, out, newErrOp(err, "cannot unmarshal JSON data from gateway"))
	}

	return c.send(ctx, out, op.Op)
}

func (c *Codec) send(ctx context.Context, ch chan<- Op, op Op) error {
	select {
	case ch <- op:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newErrOp(err error, wrap string) Op {
	if wrap != "" {
		err = errors.Wrap(err, wrap)
	}

	ev := &BackgroundErrorEvent{
		Err: err,
	}

	return Op{
		Code: ev.Op(),
		Type: ev.EventType(),
		Data: ev,
	}
}
