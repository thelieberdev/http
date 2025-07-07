package main

import (
	"fmt"
	"net"
	"log/slog"
	"os"

	"github.com/lieberdev/http/internal/request"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	sock, err := net.Listen("tcp", ":42069")
	if err != nil { slog.Error(err.Error()); return }
	defer sock.Close()
	slog.Info("Listening on port 42069")

	for {
		conn, err := sock.Accept()
		if err != nil { slog.Error(err.Error()) }
		slog.Info("Connection accepted")

		r, err := request.RequestFromReader(conn)
		if err != nil { slog.Error(err.Error()) }
		if r != nil {
			fmt.Printf("Request line:\n- Method: %s\n- Target: %s\n- Version: %s\n", 
				r.RequestLine.Method, 
				r.RequestLine.RequestTarget, 
				r.RequestLine.HttpVersion,
			)
			fmt.Printf("Headers:\n")
			for k, v := range r.Headers {
				fmt.Printf("- %s: %s\n", k, v)
			}
		}
	}
}
