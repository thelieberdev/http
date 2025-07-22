package http

import (
	"unicode"
	"io"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

// Read reads up to len(p) or numBytesPerRead bytes from the string per call
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := cr.pos + cr.numBytesPerRead
	endIndex = min(endIndex, len(cr.data))
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n
	if n > cr.numBytesPerRead {
		n = cr.numBytesPerRead
		cr.pos -= n - cr.numBytesPerRead
	}
	return n, nil
}

// Fake Close
func (cr *chunkReader) Close() error {
	return nil
}

func isUpper(s string) bool {
    for _, r := range s {
        if !unicode.IsUpper(r) && unicode.IsLetter(r) {
            return false
        }
    }
    return true
}

func grow(buf []byte) []byte {
	temp := make([]byte, len(buf)*2)
	copy(temp, buf)
	return temp
}

func getKeys[T comparable, V any](m map[T]V) []T {
    var keys []T
    for key := range m {
        keys = append(keys, key)
    }
    return keys
}
