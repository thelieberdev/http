package http

import (
	"strings"
	"bytes"
	"fmt"
)

type Headers map[string]string

func (h Headers) Get(key string) string {
	// Header names are case-insensitive
	return h[strings.ToLower(key)]
}

// Overwrites value in header
func (h *Headers) Set(key string, value string) {
	if key == "" || value == "" { return }
	name := strings.ToLower(key)
	(*h)[name] = value
}

// Adds value to header
func (h *Headers) Add(key string, value string) {
	if key == "" || value == "" { return }

	// RFC 9910 5.2
	// There can be multiple header lines with the same key
	name := strings.ToLower(key)
	if existing_value, ok := (*h)[name]; ok {
		(*h)[name] = existing_value + ", " + value
	} else {
		(*h)[name] = value
	}
}

func (h *Headers) parse(data []byte) (int, bool, error) {
	idx  := bytes.Index(data, []byte("\r\n"))
	if idx == -1 { return 0, false, nil }
	if idx == 0 { return 2, true, nil } // found end of headers, consume crlf

	parts := strings.SplitN(string(data[:idx]), ":", 2)
	if len(parts) != 2 {
		return 0, false, fmt.Errorf("Invalid header: '%s'", string(data[:idx]))
	}

	key := strings.TrimLeft(parts[0], " ")
	value := strings.TrimSpace(parts[1])
	if strings.TrimRight(key, " ") != key { 
		return 0, false, fmt.Errorf("Invalid header key: '%s'", key)
	}
	if !isValidHeaderName(key) {
		return 0, false, fmt.Errorf("Invalid character in header name: '%s'", key)
	}

	h.Add(key, value)
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
