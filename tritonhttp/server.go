package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// VirtualHosts contains a mapping from host name to the docRoot path
	// (i.e. the path to the directory to serve static files from) for
	// all virtual hosts that this server supports
	VirtualHosts map[string]string
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {
	// Hint: Validate all docRoots
	// -> Can be ignored as it is being done already in tritonhttp.ParseVHConfigFile()

	// Hint: create your listen socket and spawn off goroutines per incoming client
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// Make sure the connection closes before the function returns
	defer ln.Close()
	fmt.Println("Listening on", ln.Addr())

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println("accepted connection", conn.RemoteAddr())
		go s.handleClientConnections(conn)
	}
}

func (s *Server) handleClientConnections(conn net.Conn) {
	// Read from connection
	br := bufio.NewReader(conn)

	// Keep reading for new requests from the same connection
	for {
		// Set a read timeout
		err := conn.SetReadDeadline(time.Now().Add(CONNECT_TIMEOUT))
		if err != nil {
			conn.Close()
			return
		}

		// Read next request
		req, noReq, err := readRequest(br)
		// handle errors
		// error 1: client has closed the connection
		if errors.Is(err, io.EOF) {
			conn.Close()
			return
		}

		// error 2: Timeout from the server
		// If partial request, send `400: Client Error`
		// else, close the connection
		if err, ok := err.(net.Error); ok && err.Timeout() {
			if noReq {
				conn.Close()
				return
			}
			res := &Response{}
			res.handleBadRequest(req)
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}

		// error 3: malformed/invalid requests
		if err != nil {

			res := &Response{}
			res.handleBadRequest(req)
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}

		// Handle Good Requests
		log.Println("Handling good requests")
		res := s.handleGoodRequest(req)
		err = res.Write(conn)
		if err != nil {
			fmt.Println(err)
		}

		if req.Close {
			conn.Close()
			return
		}

	}
}
