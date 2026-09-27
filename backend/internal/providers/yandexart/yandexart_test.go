package yandexart

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/oleg3190/Web-studio-img/backend/internal/generation"
)

func TestProviderMapsAsyncOperationToImage(t *testing.T) {
	image := base64.StdEncoding.EncodeToString([]byte("jpeg"))
	operationHits := 0
	operationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		operationHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"op-1","done":true,"response":{"image":"` + image + `","modelVersion":"2026.01"}}`))
	}))
	defer operationServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/foundationModels/v1/imageGenerationAsync" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"op-1","done":false}`))
	}))
	defer apiServer.Close()

	provider, err := New(Config{
		Endpoint: apiServer.URL, OperationEndpoint: operationServer.URL,
		APIKey: "test", FolderID: "folder", PollEvery: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Generate(context.Background(), generation.Request{
		ProjectID: uuid.New(), Prompt: "hello", IdempotencyKey: "key", AspectRatio: "16:9",
	})
	if err != nil {
		t.Fatal(err)
	}
	if operationHits != 1 || len(result.Images) != 1 || string(result.Images[0].Data) != "jpeg" {
		t.Fatalf("unexpected result: hits=%d images=%d data=%q", operationHits, len(result.Images), result.Images[0].Data)
	}
}

func TestParseAspectRatio(t *testing.T) {
	w, h := parseAspectRatio("16:9")
	if w != "16" || h != "9" {
		t.Fatalf("unexpected ratio %s:%s", w, h)
	}
	if w, h := parseAspectRatio("bad"); strings.TrimSpace(w+h) != "" {
		t.Fatal("expected invalid ratio to be rejected")
	}
}
