package http

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

type State int

const (
	ParsingStatusLine State = iota
	ParsingHeaders
	ParsingBody
	Done
)

type Request struct {
	StatusLine  StatusLine
	Headers     Headers
	Body        []byte // TODO: change to io.ReadCloser
	state       State
}

const buffer_size = 8

func RequestFromReader(reader io.Reader) (*Request, error) {
	r := &Request{
		StatusLine: StatusLine{},
		Headers: Headers{},
		state: ParsingStatusLine,
	}

	unconsumed_bytes := 0
	buf := make([]byte, buffer_size)
	for r.state != Done {
		if unconsumed_bytes == len(buf) {
			// buffer is full, double size of buffer
			temp := make([]byte, len(buf)*2)
			copy(temp, buf)
			buf = temp
		}

		new_bytes, err := reader.Read(buf[unconsumed_bytes:])
		if errors.Is(err, io.EOF) { 
			if r.state != Done {
				return nil, fmt.Errorf(
					"incomplete request, reached EOF in state: %d",
					r.state,
					)
			}
			break
		} else if err != nil { 
			return nil, err 
		}
		unconsumed_bytes += new_bytes

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

func (r *Request) parse(data []byte) (int, error) {
	total_consumed_bytes := 0
	for r.state != Done {
		n, err := r.parseSingle(data[total_consumed_bytes:])
		if err != nil { return 0, err }
		if n == 0 { break } // no bytes consumed, need more data
		total_consumed_bytes += n
	}
	return total_consumed_bytes, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case ParsingStatusLine:
		consumed_bytes, err := r.StatusLine.parse(data)
		if err != nil { return 0, err }
		if consumed_bytes == 0 { return 0, nil } // no bytes consumed, need more data

		r.state = ParsingHeaders
		return consumed_bytes, nil
	case ParsingHeaders: 
		consumed_bytes, done, err := r.Headers.parse(data)
		if err != nil { return 0, err }
		if done {
			// If done parse body. If content-length is not set, then we are done
			r.state = ParsingBody 
			if r.Headers.Get("content-length") == "" { r.state = Done }
		}
		return consumed_bytes, nil
	case ParsingBody:
		r.Body = append(r.Body, data...)
		content_length, err := strconv.Atoi(r.Headers.Get("Content-Length"))
		if err != nil { return 0, err }
		if len(r.Body) > content_length {
			return 0, fmt.Errorf(
				"Body size (%d) is bigger than sent Content-Length (%d)", 
				len(r.Body), 
				content_length,
				)
		} else if len(r.Body) == content_length {
			r.state = Done
		}
		return len(data), nil
	default:
		return 0, fmt.Errorf("Request is in unknown state. Request should not be parsed")
	}
}
