package httpapi
import ("encoding/json";"log/slog";"net/http";"time")
type Server struct{handler http.Handler}
func NewServer(l *slog.Logger,origins []string)*Server{mux:=http.NewServeMux();mux.HandleFunc("GET /healthz",healthHandler);mux.HandleFunc("GET /readyz",readyHandler);mux.HandleFunc("GET /api/v1/health",apiHealthHandler);var h http.Handler=mux;h=withCORS(origins,h);h=withSecurityHeaders(h);h=withRequestID(h);h=withLogging(l,h);h=http.MaxBytesHandler(h,10<<20);return &Server{handler:h}}
func(s *Server)Handler()http.Handler{return s.handler}
func(s *Server)HTTPServer(addr string,r,w,i time.Duration)*http.Server{return &http.Server{Addr:addr,Handler:s.handler,ReadTimeout:r,WriteTimeout:w,IdleTimeout:i}}
func healthHandler(w http.ResponseWriter,_ *http.Request){writeJSON(w,200,map[string]string{"status":"ok"})}
func readyHandler(w http.ResponseWriter,_ *http.Request){writeJSON(w,200,map[string]string{"status":"ready"})}
func apiHealthHandler(w http.ResponseWriter,_ *http.Request){writeJSON(w,200,map[string]string{"status":"ok","service":"web-studio-img-api"})}
func writeJSON(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
