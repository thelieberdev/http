package request

import (
	"io"
	"bytes"
	"unicode"
	"fmt"
	"strings"
	"errors"
)

type State int

const (
	Initial State = iota
	Done
)

type Request struct {
	RequestLine RequestLine
	State       State
}

type RequestLine struct {
	Method        string
	RequestTarget string
	HttpVersion   string
}

const buffer_size = 8 // not performant

func RequestFromReader(reader io.Reader) (*Request, error) {
	r := &Request{State: Initial}
	unconsumed_bytes := 0
	buf := make([]byte, buffer_size)
	for r.State != Done {
		if unconsumed_bytes == len(buf) {
			// buffer is full, double the size of the buffer
			temp := make([]byte, len(buf)*2)
			copy(temp, buf)
			buf = temp
		}

		n, err := reader.Read(buf[unconsumed_bytes:])
		if err != nil { 
			if errors.Is(err, io.EOF) { 
				r.State = Done
				break 
			}
			return nil, err
		}
		unconsumed_bytes += n

		parsed_bytes, err := r.parse(buf[:unconsumed_bytes])
		if err != nil { return nil, err }
		if parsed_bytes != 0 { 
			// successful parsed. remove parsed bytes from buffer
			unconsumed_bytes -= parsed_bytes
			copy(buf, buf[n:])
			buf = buf[:unconsumed_bytes]
		}
	}
	return r, nil
}

func (r *Request) parse(data []byte) (int, error) {
	if r.State != Initial { 
		return 0, fmt.Errorf("Request is not in initial state. Request should not be parsed")
	}

	request_line, consumed_bytes, err := parseRequestLine(data)
	if err != nil { return 0, err }
	if consumed_bytes == 0 { return 0, nil } // no bytes consumed, need more data

	r.RequestLine = *request_line
	r.State = Done
	return consumed_bytes, nil
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 { return nil, 0, nil }

	// parts[0] = method, parts[1] = request-target, parts[2] = HTTP version
	parts := strings.Split(string(data[:idx]), " ")
	if len(parts) != 3 {
		return nil, 0, fmt.Errorf("Invalid number of parts in request line")
	}
	if !isUpper(parts[0]) {
		return nil, 0, fmt.Errorf("Request method must only conatin uppercase letters")
	}
	if parts[2] != "HTTP/1.1" {
		return nil, 0, fmt.Errorf("HTTP version must be HTTP/1.1")
	}

	return &RequestLine{parts[0], parts[1], parts[2]}, len(data), nil
}

func isUpper(s string) bool {
    for _, r := range s {
        if !unicode.IsUpper(r) && unicode.IsLetter(r) {
            return false
        }
    }
    return true
}
