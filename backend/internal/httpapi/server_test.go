package httpapi
import ("log/slog";"net/http";"net/http/httptest";"testing")
func TestHealthEndpoint(t *testing.T){s:=NewServer(slog.Default(),[]string{"http://localhost:5173"});r:=httptest.NewRequest(http.MethodGet,"/healthz",nil);w:=httptest.NewRecorder();s.Handler().ServeHTTP(w,r);if w.Code!=200{t.Fatalf("status=%d",w.Code)};if w.Header().Get("X-Content-Type-Options")!="nosniff"{t.Fatal("security header missing")}}
