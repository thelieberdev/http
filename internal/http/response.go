package http

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type ResponseStatusCode int
const (
	StatusOK ResponseStatusCode = 200
	StatusBadRequest ResponseStatusCode = 400
	StatusInternalServerError ResponseStatusCode = 500
)

type responseWriterState int
const (
	writingStatusLine responseWriterState = iota
	writingHeaders
	writingBody
	writingChunkedBody
	done
)

type ResponseWriter struct {
	Headers Headers
	Trailers Headers
	writer io.Writer
	state responseWriterState
}

func (w *ResponseWriter) WriteStatusLine(sc ResponseStatusCode) error {
	if w.state != writingStatusLine {
		return fmt.Errorf("Invalid state for writing status line: %d", w.state)
	}

	switch sc {
	case StatusOK:
		_, err := w.writer.Write([]byte("HTTP/1.1 200 OK\r\n"))
		w.state = writingHeaders
		return err
	case StatusBadRequest:
		_, err := w.writer.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		w.state = writingHeaders
		return err
	case StatusInternalServerError:
		_, err := w.writer.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n"))
		w.state = writingHeaders
		return err
	default:
		return fmt.Errorf("Invalid response status code: %d", sc)
	}
}

// Does not write headers to data. Good for adding headers after writing body
// that are necessary for the response. Such as Content-Length and Trailers
func (w *ResponseWriter) WriteHeaders(headers Headers) error {
	if w.state != writingHeaders {
		return fmt.Errorf("Invalid state for writing headers: %d", w.state)
	}

	for name, value := range headers { w.Headers.Add(name, value) }
	w.state = writingBody
	return nil
}

func (w *ResponseWriter) WriteTrailers(headers Headers) error {
	if w.state != writingBody {
		return fmt.Errorf("Invalid state for writing trailers: %d", w.state)
	}

	w.Headers.Add("Trailer", strings.Join(getKeys(headers), " "))
	w.Trailers = headers

	return nil
}

func (w *ResponseWriter) WriteBody(data []byte) (int, error) {
	if w.state != writingBody {
		return 0, fmt.Errorf("Invalid state for writing body: %d", w.state)
	}
	if _, ok := w.Headers["content-type"]; !ok {
		return 0, fmt.Errorf("Content-Type header is required to write to body")
	}

	// Set default headers
	if _, ok := w.Headers["transfer-encoding"]; !ok {
		w.Headers.Set("Content-Length", strconv.Itoa(len(data)))
	}
	if _, ok := w.Headers["connection"]; !ok {
		w.Headers.Set("Connection", "close")
	}

	total_written := 0

	// Write headers
	for name, value := range w.Headers {
		n, err := w.writer.Write([]byte(name + ": " + value + "\r\n"))
		if err != nil { return 0, err }
		total_written += n
	}
	n, err := w.writer.Write([]byte("\r\n"))
	if err != nil { return 0, err }
	total_written += n
	
	// Write body
	n, err = w.writer.Write(data)
	if err != nil { return 0, err }
	total_written += n

	// Write trailers
	for name, value := range w.Headers {
		n, err := w.writer.Write([]byte(name + ": " + value + "\r\n"))
		if err != nil { return 0, err }
		total_written += n
	}
	n, err = w.writer.Write([]byte("\r\n"))
	if err != nil { return 0, err }
	total_written += n

	w.state = done
	return total_written, nil
}

func (w *ResponseWriter) WriteChunkedBody(data []byte) (int, error) {
	if w.state != writingBody && w.state != writingChunkedBody {
		return 0, fmt.Errorf("Invalid state for writing body: %d", w.state)
	}
	if enc := w.Headers.Get("transfer-encoding"); enc != "chunked" {
		return 0, fmt.Errorf("Transfer-Encoding must be set to chunked to write chunked body")
	}

	total_written := 0

	// Write headers once
	if w.state == writingBody {
		for name, value := range w.Headers {
			n, err := w.writer.Write([]byte(name + ": " + value + "\r\n"))
			if err != nil { return 0, err }
			total_written += n
		}
		n, err := w.writer.Write([]byte("\r\n"))
		if err != nil { return 0, err }
		total_written += n
	}

	// Write data len in hex
	hex := strconv.FormatInt(int64(len(data)), 16)
	w.writer.Write([]byte(hex + "\r\n"))
	w.writer.Write(data)
	w.writer.Write([]byte("\r\n"))

	w.state = writingChunkedBody

	return total_written, nil
}

func (w *ResponseWriter) WriteChunkedBodyDone() (int, error) {
	if w.state != writingChunkedBody {
		return 0, fmt.Errorf("Invalid state for writing chunked body done: %d", w.state)
	}

	total_written := 0

	// Write chunked body end
	n, err := w.writer.Write([]byte("0\r\n"))
	if err != nil { return 0, err }
	total_written += n

	// Write trailers
	for name, value := range w.Headers {
		n, err := w.writer.Write([]byte(name + ": " + value + "\r\n"))
		if err != nil { return 0, err }
		total_written += n
	}
	n, err = w.writer.Write([]byte("\r\n"))
	if err != nil { return 0, err }
	total_written += n

	w.state = done

	return total_written, nil
}
