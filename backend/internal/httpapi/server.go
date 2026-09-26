package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	handler http.Handler
}

func NewServer(logger *slog.Logger, origins []string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readyHandler)
	mux.HandleFunc("GET /api/v1/health", apiHealthHandler)

	var handler http.Handler = mux
	handler = withCORS(origins, handler)
	handler = withSecurityHeaders(handler)
	handler = withRequestID(handler)
	handler = withLogging(logger, handler)
	handler = http.MaxBytesHandler(handler, 10<<20)

	return &Server{handler: handler}
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) HTTPServer(addr string, readTimeout, writeTimeout, idleTimeout time.Duration) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      s.handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func apiHealthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "web-studio-img-api"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
