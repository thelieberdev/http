package request

import (
	"io"
	"bytes"
	"unicode"
	"fmt"
	"strings"
	"errors"

	"github.com/lieberdev/http/internal/headers"
)

type State int

const (
	Initial State = iota
	ParsingHeaders
	Done
)

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	State       State
}

type RequestLine struct {
	Method        string
	RequestTarget string
	HttpVersion   string
}

const buffer_size = 8 // not performant

func RequestFromReader(reader io.Reader) (*Request, error) {
	r := &Request{
		Headers: headers.NewHeaders(),
		State: Initial,
	}

	unconsumed_bytes := 0
	buf := make([]byte, buffer_size)
	for r.State != Done {
		if unconsumed_bytes >= len(buf) {
			// buffer is full, double the size of the buffer
			temp := make([]byte, len(buf)*2)
			copy(temp, buf)
			buf = temp
		}

		n, err := reader.Read(buf[unconsumed_bytes:])
		if err != nil { 
			if errors.Is(err, io.EOF) { 
				if r.State != Done {
					return nil, fmt.Errorf(
							"incomplete request, in state: %d, read n bytes on EOF: %d",
							r.State, 
							unconsumed_bytes,
						)
				}
				break
			}
			return nil, err
		}
		unconsumed_bytes += n

		parsed_bytes, err := r.parse(buf[:unconsumed_bytes])
		if err != nil { return nil, err }
		if parsed_bytes != 0 { 
			// successful parsed. remove parsed bytes from buffer
			copy(buf, buf[parsed_bytes:])
			unconsumed_bytes -= parsed_bytes
			// buf = buf[:unconsumed_bytes]
		}
	}
	return r, nil
}

var count = 0

func (r *Request) parse(data []byte) (int, error) {
	total_consumed_bytes := 0
	for r.State != Done {
		n, err := r.parseSingle(data[total_consumed_bytes:])
		if err != nil { return 0, err }
		if n == 0 { break } // no bytes consumed, need more data
		total_consumed_bytes += n
	}
	return total_consumed_bytes, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.State {
	case Initial:
		request_line, consumed_bytes, err := parseRequestLine(data)
		if err != nil { return 0, err }
		if consumed_bytes == 0 { return 0, nil } // no bytes consumed, need more data

		r.RequestLine = *request_line
		r.State = ParsingHeaders
		return consumed_bytes, nil
	case ParsingHeaders: 
		consumed_bytes, done, err := r.Headers.Parse(data)
		if err != nil { return 0, err }
		if done { r.State = Done }
		return consumed_bytes, nil
	default:
		return 0, fmt.Errorf("Request is not in initial or parsing headers state. Request should not be parsed")
	}
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

	return &RequestLine{parts[0], parts[1], parts[2]}, idx + 2, nil
}

func isUpper(s string) bool {
    for _, r := range s {
        if !unicode.IsUpper(r) && unicode.IsLetter(r) {
            return false
        }
    }
    return true
}
