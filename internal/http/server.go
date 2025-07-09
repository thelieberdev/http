package http

import (
	"net"
	"strconv"
	"sync/atomic"
	"log"
	"log/slog"
	"os"
)

type Server struct {
	Listener net.Listener
	Handler Handler
	ErrorLog *log.Logger
	closed atomic.Bool
}

type Handler func(w ResponseWriter, req *Request)

func ListenAndServe(address string, handler Handler) (*Server, error) {
	if address == "" { address = ":http" }
	ln, err := net.Listen("tcp", address)
	if err != nil { return nil, err }

	srv := &Server{
		Listener: ln,
		Handler: handler,
		ErrorLog: slog.NewLogLogger(slog.NewJSONHandler(os.Stderr, nil), slog.LevelError),
	}
    
	go srv.Serve()
	return srv, nil
}

func (s *Server) Serve() error {
	for {
		conn, err := s.Listener.Accept()
		if s.closed.Load() {
			return nil // Graceful exit
		}
		if err != nil {
			s.ErrorLog.Printf("Error: %v", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.Listener.Close()
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	w := ResponseWriter{
		Headers: Headers{},
		writer: conn,
		state: writingStatusLine,
	}

	r, err := RequestFromReader(conn)
	if err != nil {
		w.WriteStatusLine(StatusBadRequest)
		w.WriteHeaders(Headers{
			"Connection": "close",
			"Content-Type": "text/plain", // TODO: Add possibility to set content type
			"Content-Length": strconv.Itoa(len([]byte(err.Error()))),
		})
		w.WriteBody([]byte(err.Error()))
		return 
	}

	s.Handler(w, r)
}
