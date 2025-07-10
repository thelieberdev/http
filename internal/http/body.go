package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
)

type chunkedBody struct {
	rc io.ReadCloser
	buf []byte
	unconsumed_bytes int
	next_chunk_size int64
	eof bool
	closed atomic.Bool
}

func newChunkedBody(rc io.ReadCloser) *chunkedBody {
	return &chunkedBody{
		rc: rc,
		buf: make([]byte, buffer_size),
		unconsumed_bytes: 0,
		next_chunk_size: -1,
	}
}

func (cb *chunkedBody) Read(p []byte) (int, error) {
	if cb.closed.Load() { return 0, io.ErrClosedPipe }

	for cb.unconsumed_bytes < len(p) && !cb.eof {
		if len(cb.buf) == cb.unconsumed_bytes {
			// buffer is full, double size of buffer
			temp := make([]byte, len(cb.buf)*2)
			copy(temp, cb.buf)
			cb.buf = temp
		}

		n, err := cb.rc.Read(cb.buf[cb.unconsumed_bytes:])
		if errors.Is(err, io.EOF) { 
			cb.eof = true
			break
		} else if err != nil { 
			return 0, err
		}
		cb.unconsumed_bytes += n

		consumed_bytes, err := cb.parse(&p)
		if err != nil { return 0, err }
		if consumed_bytes != 0 {
			copy(cb.buf, cb.buf[consumed_bytes:])
			cb.unconsumed_bytes -= consumed_bytes
		}

		return consumed_bytes, nil
	}
	
	if cb.unconsumed_bytes == 0 { return 0, io.EOF }
	n := min(len(cb.buf[:cb.unconsumed_bytes]), len(p))
	copy(p, cb.buf[:n])
	copy(cb.buf, cb.buf[n:])
	cb.unconsumed_bytes -= n
	return n, nil
}

func (cb *chunkedBody) parse(p *[]byte) (int, error) {
	idx := bytes.Index(cb.buf, []byte("\r\n"))
	if idx == -1 { return 0, nil } // need more data

  // -1 is a sentinal value for unknown chunk size
	if cb.next_chunk_size == -1 {
		size, err := strconv.ParseInt(string(cb.buf[:idx]), 16, 64)
		if err != nil { return 0, err }
		cb.next_chunk_size = size
		return idx + 2, nil
	}
	
	if int64(len(cb.buf[:cb.unconsumed_bytes])) != cb.next_chunk_size {
		return 0, fmt.Errorf(
			"Sent chunk size (%d) is different than chunk (%d)", 
			cb.next_chunk_size,
			len(cb.buf[:cb.unconsumed_bytes]),
			)
	}

	n := min(len(cb.buf[:idx]), len(*p))
	copy(*p, cb.buf[:n])
	return n + 2, nil
}

func (cb *chunkedBody) Close() error {
	cb.closed.Store(true)
	return cb.rc.Close()
}
