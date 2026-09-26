package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: errorBody{
		Code: code, Message: message, RequestID: requestIDFromContext(r.Context()),
	}})
}

func requestIDFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(requestIDKey{}).(string); ok && value != "" {
		return value
	}
	return uuid.NewString()
}
