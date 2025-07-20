package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
)

type chunkedBodyState int
const (
	ReadingChunkSize chunkedBodyState = iota
	ReadingChunk
)

type body struct {
	rc io.ReadCloser
	buf []byte
	unconsumed_bytes int
	content_length int
	closed atomic.Bool
	eof bool
	total_consumed_bytes int
	// Needed for chunked body parsing
	is_chunked bool
	chunk_size int
	consumed_chunk_bytes int
	cb_state chunkedBodyState
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

	if b.is_chunked {
		consumed_bytes, err := b.parseChunkedBody(p)
		if err != nil { return 0, err }
		if consumed_bytes != 0 {
			copy(p, b.buf[:consumed_bytes])
			copy(b.buf, b.buf[consumed_bytes:])
			b.unconsumed_bytes -= consumed_bytes
		}
		return consumed_bytes, nil
	} else {
		// uncosmumed_bytes can still be smaller if EOF was reached
		consumed_bytes := min(b.unconsumed_bytes, len(p))
		// Also do not consume more bytes than the content length
		remaining_bytes := b.content_length - b.total_consumed_bytes
		if remaining_bytes <= consumed_bytes {
			consumed_bytes = remaining_bytes
			b.eof = true
		}
		copy(p, b.buf[:consumed_bytes])
		copy(b.buf, b.buf[consumed_bytes:])
		b.unconsumed_bytes -= consumed_bytes
		b.total_consumed_bytes += consumed_bytes
		return consumed_bytes, nil
	}
}

func (b *body) parseChunkedBody(p []byte) (int, error) {
	idx := bytes.Index(b.buf, []byte("\r\n"))
	if idx == -1 { return 0, nil } // need more data

	switch b.cb_state {
	case ReadingChunkSize:
		size, err := strconv.ParseInt(string(b.buf[:idx]), 16, 64)
		if err != nil { return 0, err }
		b.chunk_size = int(size)
		b.cb_state = ReadingChunk
		return idx + 2, nil
	case ReadingChunk:
		remaining_bytes := b.chunk_size - b.consumed_chunk_bytes
		if len(b.buf[:idx]) != remaining_bytes {
			return 0, fmt.Errorf("Sent chunk size is different than chunk len")
		}
		// Read only part of chunk
		if len(p) < remaining_bytes {
			b.consumed_chunk_bytes += len(p)
			return len(p), nil
		}
		b.cb_state = ReadingChunkSize
		return remaining_bytes + 2, nil
	default:
		return 0, fmt.Errorf("Unknown chunked body state: %d", b.cb_state)
	}
}

func (b *body) Close() error {
	b.closed.Store(true)
	return b.rc.Close()
}
