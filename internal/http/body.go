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
	readChunkSize chunkedBodyState = iota
	readChunk
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

func (b *body) Read(data []byte) (int, error) {
	if b.closed.Load() { return 0, io.ErrClosedPipe }
	if b.eof { return 0, io.EOF }

	for {
		// Check if buffer is full
		if len(b.buf) == b.unconsumed_bytes { b.buf = grow(b.buf) }

		n, err := b.rc.Read(b.buf[b.unconsumed_bytes:])
		if errors.Is(err, io.EOF) {
			b.eof = true
			break
		} else if err != nil { 
			return 0, err
		}
		b.unconsumed_bytes += n

		consumed_bytes := 0
		err = error(nil)
		if b.is_chunked {
			consumed_bytes, err = b.parseChunked(data)
		} else {
			consumed_bytes, err = b.parseFixed(data)
		}
		if err != nil { return 0, err }
		if consumed_bytes != 0 { 
			return consumed_bytes, nil
		}
	}
	return 0, nil
}

func (b *body) parseFixed(data []byte) (int, error) {
	// uncosmumed_bytes can still be smaller if EOF was reached
	consumed_bytes := min(b.unconsumed_bytes, len(data))
	// Also do not consume more bytes than the content length
	remaining_bytes := b.content_length - b.total_consumed_bytes
	if remaining_bytes <= consumed_bytes {
		consumed_bytes = remaining_bytes
		b.eof = true
	}
	copy(data, b.buf[:consumed_bytes])
	copy(b.buf, b.buf[consumed_bytes:])
	b.unconsumed_bytes -= consumed_bytes
	b.total_consumed_bytes += consumed_bytes
	return consumed_bytes, nil
}

func (b *body) parseChunked(data []byte) (int, error) {
	idx := bytes.Index(b.buf[:b.unconsumed_bytes], []byte("\r\n"))
	if idx == -1 { return 0, nil }

	switch b.cb_state {
	case readChunkSize:
		size, err := strconv.ParseInt(string(b.buf[:idx]), 16, 64)
		if err != nil { return 0, err }
		b.chunk_size = int(size)
		b.cb_state = readChunk
		copy(b.buf, b.buf[idx+2:])
		b.unconsumed_bytes -= idx+2
		return 0, nil
	case readChunk:
		remaining_bytes := b.chunk_size - b.consumed_chunk_bytes
		if remaining_bytes != idx {
			return 0, fmt.Errorf("Sent chunk size is different than chunk len")
		}

		// Read only part of chunk
		if len(data) < remaining_bytes {
			copy(data, b.buf[:len(data)])
			copy(b.buf, b.buf[len(data):])
			b.unconsumed_bytes -= len(data)
			b.consumed_chunk_bytes += len(data)
			return len(data), nil
		} else {
			copy(data, b.buf[:remaining_bytes])
			copy(b.buf, b.buf[remaining_bytes+2:])
			b.unconsumed_bytes -= remaining_bytes + 2
			// Finished consuming chunk
			b.consumed_chunk_bytes = 0
			b.cb_state = readChunkSize
			return remaining_bytes, nil
		}
	default:
		return 0, fmt.Errorf("Unknown chunked body state: %d", b.cb_state)
	}
}

func (b *body) Close() error {
	b.closed.Store(true)
	return b.rc.Close()
}
