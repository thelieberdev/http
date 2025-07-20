package http

import (
	"errors"
	"io"
	"sync/atomic"
)

type body struct {
	rc io.ReadCloser
	buf []byte
	unconsumed_bytes int
	content_length int
	closed atomic.Bool
	eof bool
	total_consumed_bytes int
}

func (b *body) Read(p []byte) (int, error) {
	if b.closed.Load() { return 0, io.ErrClosedPipe }
	if b.eof { return 0, io.EOF }

	for b.unconsumed_bytes < len(p) {
		if len(b.buf) == b.unconsumed_bytes {
			// buffer is full, double size of buffer
			temp := make([]byte, len(b.buf)*2)
			copy(temp, b.buf)
			b.buf = temp
		}

		n, err := b.rc.Read(b.buf[b.unconsumed_bytes:])
		if errors.Is(err, io.EOF) {
			b.eof = true
			break
		} else if err != nil { 
			return 0, err
		}
		b.unconsumed_bytes += n
	}

	// uncosmumed_bytes can still be smaller if EOF was reached
	consumed_bytes := min(b.unconsumed_bytes, len(p))
	// Also do not consume more bytes than the content length
	remaining_bytes := b.content_length - b.total_consumed_bytes
	if remaining_bytes <= consumed_bytes {
		consumed_bytes = remaining_bytes
		b.eof = true
	}
	copy(p, b.buf[:consumed_bytes])
	b.unconsumed_bytes -= consumed_bytes
	b.total_consumed_bytes += consumed_bytes
	return consumed_bytes, nil
}

func (b *body) Close() error {
	b.closed.Store(true)
	return b.rc.Close()
}
