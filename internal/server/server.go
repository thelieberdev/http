package server

import (
	"bytes"
	"io"
	"net"
	"sync/atomic"

	"github.com/lieberdev/http/internal/request"
	"github.com/lieberdev/http/internal/response"
)

type Server struct {
	address string
	shutdown atomic.Bool
	handler Handler
}

type Handler func(w io.Writer, req *request.Request) *HandlerError

type HandlerError struct {
	StatusCode response.StatusCode
	Message string
}

func Serve(address string, handler Handler) (*Server, error) {
	server := &Server{
		address: address,
		handler: handler,
	}
	go server.Listen()
	return server, nil
}

func (s *Server) Close() {
	s.shutdown.Store(true)
}

func (s *Server) Listen() error {
	addr := s.address
	if addr == "" { addr = ":http" }
	ln, err := net.Listen("tcp", addr)
	if err != nil { return err }
	defer ln.Close()

	for !s.shutdown.Load() {
		conn, err := ln.Accept()
		if err != nil { return err }
		go s.handle(conn)
	}

	return nil
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	r, err := request.RequestFromReader(conn)
	if err != nil { return }

	buf := bytes.NewBuffer(nil)
	h_err := s.handler(buf, r)
	if h_err != nil {
		response.WriteStatusLine(conn, h_err.StatusCode)
		response.WriteHeaders(conn, response.GetDefaultHeaders(len([]byte(h_err.Message))))
		conn.Write([]byte(h_err.Message))
		return
	}

	// Write status line
	err = response.WriteStatusLine(conn, response.StatusOK)
	if err != nil { return }
    
	// Write headers
	err = response.WriteHeaders(conn, response.GetDefaultHeaders(buf.Len()))
	if err != nil { return }

	// Write body
  _, err = conn.Write(buf.Bytes())
	if err != nil { return }
}
