package identd

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// Server implements a simple RFC 1413 identd server
type Server struct {
	listener net.Listener
	username string
	port     string
}

// New creates a new identd server
func New(port, username string) *Server {
	return &Server{
		port:     port,
		username: username,
	}
}

// Start begins listening for identd requests
func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("failed to start identd server: %w", err)
	}

	log.Printf("Identd server listening on port %s", s.port)

	go s.acceptConnections()
	return nil
}

// Stop stops the identd server
func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Listener closed, exit gracefully
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set a reasonable timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	reader := bufio.NewReader(conn)
	request, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	request = strings.TrimSpace(request)
	parts := strings.Split(request, ",")

	if len(parts) != 2 {
		return
	}

	serverPort := strings.TrimSpace(parts[0])
	clientPort := strings.TrimSpace(parts[1])

	// RFC 1413 response format: <server-port>, <client-port> : USERID : UNIX : <username>
	response := fmt.Sprintf("%s, %s : USERID : UNIX : %s\r\n", serverPort, clientPort, s.username)
	conn.Write([]byte(response))
}
