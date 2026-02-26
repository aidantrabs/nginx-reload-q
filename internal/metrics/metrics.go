package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aidantrabs/nginx-reload-q/internal/queue"
)

type Server struct {
	q   *queue.Queue
	log *slog.Logger
	srv *http.Server
}

func NewServer(addr string, q *queue.Queue, log *slog.Logger) *Server {
	s := &Server{q: q, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /metrics", s.handleMetrics)

	s.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s
}

func (s *Server) ListenAndServe() error {
	s.log.Info("metrics listening", "addr", s.srv.Addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Close() error {
	return s.srv.Close()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}` + "\n"))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(s.q.Stats()); err != nil {
		s.log.Error("failed to write metrics", "err", err)
	}
}
