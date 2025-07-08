package response

import (
	"errors"
	"io"
	"strconv"

	"github.com/lieberdev/http/internal/headers"
)

type StatusCode int

const (
	StatusOK StatusCode = 200
	StatusBadRequest StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	switch statusCode {
	case StatusOK:
		_, err := w.Write([]byte("HTTP/1.1 200 OK\r\n"))
		return err
	case StatusBadRequest:
		_, err := w.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		return err
	case StatusInternalServerError:
		_, err := w.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n"))
		return err
	default:
		return errors.New("invalid status code")
	}
}

func GetDefaultHeaders(bodySize int) headers.Headers {
	headers := headers.NewHeaders()

	headers.Set("Content-Length", strconv.Itoa(bodySize))
	headers.Set("Content-Type", "text/plain")
	headers.Set("Connection", "close")

	return headers
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	for name, value := range headers {
		_, err := w.Write([]byte(name + ": " + value + "\r\n"))
		if err != nil { return err }
	}
	_, err := w.Write([]byte("\r\n"))
	if err != nil { return err }
	return nil
}
