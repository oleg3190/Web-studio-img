package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/oleg3190/Web-studio-img/backend/internal/iterations"
	"github.com/oleg3190/Web-studio-img/backend/internal/generation"
	"github.com/oleg3190/Web-studio-img/backend/internal/projects"
	"github.com/oleg3190/Web-studio-img/backend/internal/security"
)

type Server struct {
	handler http.Handler
}

func NewServer(logger *slog.Logger, origins []string, limiter *security.RateLimiter) *Server {
	return newServer(logger, origins, limiter, nil, nil, nil)
}

func NewServerWithProjects(logger *slog.Logger, origins []string, limiter *security.RateLimiter, projectHandler *projects.Handler) *Server {
	return newServer(logger, origins, limiter, projectHandler, nil, nil)
}

func NewServerWithProjectsAndIterations(logger *slog.Logger, origins []string, limiter *security.RateLimiter, projectHandler *projects.Handler, iterationHandler *iterations.Handler) *Server {
	return newServer(logger, origins, limiter, projectHandler, iterationHandler, nil)
}

func NewServerWithProjectsIterationsAndGeneration(logger *slog.Logger, origins []string, limiter *security.RateLimiter, projectHandler *projects.Handler, iterationHandler *iterations.Handler, generationHandler *generation.Handler) *Server {
	return newServer(logger, origins, limiter, projectHandler, iterationHandler, generationHandler)
}

func newServer(logger *slog.Logger, origins []string, limiter *security.RateLimiter, projectHandler *projects.Handler, iterationHandler *iterations.Handler, generationHandler *generation.Handler) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readyHandler)
	mux.HandleFunc("GET /api/v1/health", apiHealthHandler)
	if projectHandler != nil {
		projectHandler.Register(mux)
	}
	if iterationHandler != nil {
		iterationHandler.Register(mux)
	}
	if generationHandler != nil {
		generationHandler.Register(mux)
	}

	var handler http.Handler = mux
	handler = withCORS(origins, handler)
	handler = withSecurityHeaders(handler)
	handler = withRequestID(handler)
	handler = withRateLimit(limiter, handler)
	handler = withLogging(logger, handler)
	handler = http.MaxBytesHandler(handler, security.DefaultMaxUploadSize)

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
