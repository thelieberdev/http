package headers

import (
	"bytes"
	"strings"
	"fmt"
)

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

func (h Headers) Get(key string) string {
	return h[strings.ToLower(key)]
}

func (h Headers) Set(key string, value string) {
	if current_value, ok := h[key]; ok {
		h[key] = current_value + ", " + value
	} else {
		h[key] = value
	}
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx  := bytes.Index(data, []byte("\r\n"))
	if idx == -1 { return 0, false, nil }
	if idx == 0 { return 2, true, nil } // found end of headers, consume crlf

	parts := bytes.SplitN(data[:idx], []byte(":"), 2)
	if len(parts) != 2 {
		return 0, false, fmt.Errorf("Invalid header: '%s'", string(data[:idx]))
	}

	key := strings.TrimLeft(strings.ToLower(string(parts[0])), " ")
	if strings.TrimRight(key, " ") != key { 
		return 0, false, fmt.Errorf("Invalid header key: '%s'", key)
	}
	value := strings.TrimSpace(string(parts[1]))

	if !isValidHeaderName(key) {
		return 0, false, fmt.Errorf("Invalid character in header name: '%s'", key)
	}

	h.Set(key, value)
	return idx + 2, false, nil
}

// See RFC 9910 5.1 and 5.6.2
func isValidHeaderName(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '!' || r == '#' || r == '$' || r == '%' || r == '&' ||
			r == '\'' || r == '*' || r == '+' || r == '-' || r == '.' ||
			r == '^' || r == '_' || r == '`' || r == '|' || r == '~':
			continue
		default:
			return false
		}
	}
	return true
}
