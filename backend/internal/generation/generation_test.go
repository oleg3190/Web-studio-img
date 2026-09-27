package generation

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateRequestRequiresIdempotencyAndPrompt(t *testing.T) {
	req := Request{ProjectID: uuid.New(), Prompt: "prompt"}
	if err := ValidateRequest(req); err == nil {
		t.Fatal("expected missing idempotency key to fail")
	}
	req.IdempotencyKey = "key"
	if err := ValidateRequest(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequestRejectsOversizedPrompt(t *testing.T) {
	req := Request{ProjectID: uuid.New(), Prompt: string(make([]byte, 20001)), IdempotencyKey: "key"}
	if err := ValidateRequest(req); err == nil {
		t.Fatal("expected oversized prompt to fail")
	}
}
