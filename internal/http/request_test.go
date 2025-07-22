package http

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusLineParseFromReader(t *testing.T) {
	// Test: Good GET Request line
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.StatusLine.Method)
	assert.Equal(t, "/", r.StatusLine.Target)
	assert.Equal(t, "HTTP/1.1", r.StatusLine.Version)

	// Test: Good GET Request line with path
	reader = &chunkReader{
		data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.StatusLine.Method)
	assert.Equal(t, "/coffee", r.StatusLine.Target)
	assert.Equal(t, "HTTP/1.1", r.StatusLine.Version)

	// Test: Good POST Request line with path
	reader = &chunkReader{
		data:            "POST /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/8.7.1\r\nAccept: */*\r\nContent-Type: application/json\r\nContent-Length: 22\r\n\r\n{\"flavor\":\"dark mode\"}",
		numBytesPerRead: 43,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.StatusLine.Method)
	assert.Equal(t, "/coffee", r.StatusLine.Target)
	assert.Equal(t, "HTTP/1.1", r.StatusLine.Version)

	// Test: Invalid number of parts in request line
	reader = &chunkReader{
		data:            "/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 82, // number of bytes is equal to the length of the request line
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Invalid method (out of order) Request line
	reader = &chunkReader{
		data:            "HTTP/1.1 GET /\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 4,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Invalid version in Request line
	reader = &chunkReader{
		data:            "GET /coffee HTTP/2\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 1,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)
}

func TestHeadersParseFromReader(t *testing.T) {
	// Test: Standard Headers
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 50,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "localhost:42069", r.Headers.Get("host"))
	assert.Equal(t, "curl/7.81.0", r.Headers.Get("user-agent"))
	assert.Equal(t, "*/*", r.Headers.Get("accept"))

	// Test: Empty Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Empty(t, r.Headers)

	// Test: Malformed Header
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
		numBytesPerRead: 10,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Duplicate Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nX-Test: first\r\nX-Test: second\r\n\r\n",
		numBytesPerRead: 20,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "first, second", r.Headers.Get("X-Test"))

	// Test: Case Insensitive Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nX-Mixed-Case: Value1\r\nx-mixed-case: Value2\r\nX-MIXED-CASE: Value3\r\n\r\n",
		numBytesPerRead: 5,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "Value1, Value2, Value3", r.Headers.Get("x-mixed-case"))

	// Test: Missing End of Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost\r\n",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)
}

func TestBodyParseFromReader(t *testing.T) {
	// Test: Standard Body
	reader := &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"Content-Length: 13\r\n" +
					"\r\n" +
					"hello world!\n",
		numBytesPerRead: 13,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, "hello world!\n", string(body))

	// Test: Empty Body, 0 reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"Content-Length: 0\r\n" +
					"\r\n",
		numBytesPerRead: 20,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	body, err = io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Empty(t, body)

	// Test: Empty Body, no reported content length
	reader = &chunkReader{
		data: "GET / HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"\r\n",
		numBytesPerRead: 40,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	body, err = io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Empty(t, body)

	// Test: Body bigger than reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"Content-Length: 2\r\n" +
					"\r\n" +
					"partial content",
		numBytesPerRead: 8,
	}
	r, err = RequestFromReader(reader)
	body, err = io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, "pa", string(body))

	// Test: No Content-Length but Body Exists
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
		      "Host: localhost:42069\r\n" +
		      "\r\n" +
		      "unexpected body",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	body, err = io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Empty(t, body)
}

func TestChunkedBodyParseFromReader(t *testing.T) {
	// Test: Standard Chunked Body
	t.Run("Standard Chunked Body", func(t *testing.T) {
		reader := &chunkReader{
			data: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Transfer-Encoding: chunked\r\n" +
				"\r\n" +
				"d\r\n" +
				"hello world!\n\r\n" +
				"5\r\n" +
				"-more\r\n" +
				"0\r\n" +
				"\r\n",
			numBytesPerRead: 2,
		}
		r, err := RequestFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, r)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello world!\n-more", string(body))
	})
}
