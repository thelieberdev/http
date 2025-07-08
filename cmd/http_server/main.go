package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"io"

	"github.com/lieberdev/http/internal/response"
	"github.com/lieberdev/http/internal/request"
	"github.com/lieberdev/http/internal/server"
)

const port = ":42069"

func main() {
	handler := func(w io.Writer, r *request.Request) *server.HandlerError {
		if r.RequestLine.RequestTarget == "/yourproblem" {
			return &server.HandlerError{
				StatusCode: response.StatusBadRequest,
				Message: "Your problem is not my problem\n",
			}
		} else if r.RequestLine.RequestTarget == "/myproblem" {
			return &server.HandlerError{
				StatusCode: response.StatusInternalServerError,
				Message: "Woopsie, my bad\n",
			}
		}

		return &server.HandlerError{
			StatusCode: response.StatusOK,
			Message: "All good, frfr\n",
		}
	}

	server, err := server.Serve(port, handler)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer server.Close()
	slog.Info("Server started", 
		"port", port,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	slog.Info("Server gracefully stopped")
}
