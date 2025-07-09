package http

import (
	"bytes"
	"fmt"
	"strings"
)

type StatusLine struct {
	Method        string
	Target        string
	Version       string
}

func (sl *StatusLine) parse(data []byte) (int, error) {
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 { return 0, nil }

	// parts[0] = method, parts[1] = request-target, parts[2] = HTTP version
	parts := strings.Split(string(data[:idx]), " ")
	if len(parts) != 3 {
		return 0, fmt.Errorf("Status line must only have 3 space-separated parts")
	}

	if !isUpper(parts[0]) {
		return 0, fmt.Errorf("Request-method must only contain uppercase letters")
	}
	if !strings.HasPrefix(parts[1], "/") {
		return 0, fmt.Errorf("Request-target must start with a slash")
	}
	if parts[2] != "HTTP/1.1" {
		return 0, fmt.Errorf("HTTP version must be HTTP/1.1")
	}

	sl.Method = parts[0]
	sl.Target = parts[1]
	sl.Version = parts[2]

	return idx + 2, nil
}
