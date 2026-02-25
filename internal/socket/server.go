package socket

import (
	"fmt"
	"log/slog"
	"net"
	"os"
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

// listen creates the UDS and starts accepting connections
// removes any stale socket file before binding
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

// close shuts down the listener and removes the socket file
func (s *Server) Close() error {
	if s.listener != nil {
		s.listener.Close()
	}

	os.Remove(s.path)
	s.log.Info("socket closed", "path", s.path)

	return nil
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
