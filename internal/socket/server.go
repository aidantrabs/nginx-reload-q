package socket

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
)

type Server struct {
	path     string
	listener net.Listener
	log      *slog.Logger
}

func NewServer(path string, log *slog.Logger) *Server {
	return &Server{
		path: path,
		log:  log,
	}
}

// creates the UDS and starts accepting connections
func (s *Server) Listen() error {
	if err := removeStaleSocket(s.path); err != nil {
		return fmt.Errorf("removing stale socket: %w", err)
	}

	ln, err := net.Listen("unix", s.path)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", s.path, err)
	}

	if err := os.Chmod(s.path, 0600); err != nil {
		ln.Close()
		return fmt.Errorf("setting socket permissions: %w", err)
	}

	s.listener = ln
	s.log.Info("socket listening", "path", s.path)

	return nil
}

// shuts down the listener and removes the socket file
func (s *Server) Close() error {
	if s.listener != nil {
		s.listener.Close()
	}

	os.Remove(s.path)
	s.log.Info("socket closed", "path", s.path)

	return nil
}

// blocks and handles incoming connections
func (s *Server) Accept() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		fmt.Fprintf(conn, "ERROR: empty request\n")
		return
	}

	cmd := strings.TrimSpace(scanner.Text())

	switch cmd {
	case "RELOAD":
		s.log.Info("reload requested")
		fmt.Fprintf(conn, "OK\n")
	default:
		s.log.Warn("unknown command", "cmd", cmd)
		fmt.Fprintf(conn, "ERROR: unknown command\n")
	}
}

func removeStaleSocket(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return os.Remove(path)
}
