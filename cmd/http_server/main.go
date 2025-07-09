package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lieberdev/http/internal/http"
)

const port = ":42069"

func main() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.StatusLine.Target == "/yourproblem" {
			w.WriteStatusLine(http.StatusBadRequest)
			w.Headers.Set("Content-Type", "text/html")
			w.WriteHeaders(nil)
			w.WriteBody([]byte(`<html>
				<head>
				<title>400 Bad Request</title>
				</head>
				<body>
				<h1>This is your problem!</h1>
				<p>Your problem is not my problem.</p>
				</body>
				</html>`))
			return
		} else if r.StatusLine.Target == "/myproblem" {
			w.WriteStatusLine(http.StatusInternalServerError)
			w.Headers.Set("Content-Type", "text/html")
			w.WriteHeaders(nil)
			w.WriteBody([]byte(`<html>
				<head>
				<title>500 Internal Server Error</title>
				</head>
				<body>
				<h1>This is my problem!</h1>
				<p>Woopsie, my bad</p>
				</body>
				</html>`))
			return
		}

		w.WriteStatusLine(http.StatusOK)
		w.Headers.Set("Content-Type", "text/html")
		w.WriteHeaders(nil)
		w.WriteBody([]byte(`<html>
			<head>
			<title>200 OK</title>
			</head>
			<body>
			<h1>Success!</h1>
			<p>All good, frfr</p>
			</body>
			</html>`))
	}

	server, err := http.ListenAndServe(port, handler)
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
