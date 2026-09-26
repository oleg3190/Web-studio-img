package queue

import (
	"context"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(Config{}); err == nil {
		t.Fatal("expected invalid config error")
	}
	if _, err := NewClient(Config{Address: "localhost:6379", DB: -1}); err == nil {
		t.Fatal("expected negative db error")
	}
}

func TestNewClientAndEnqueueValidation(t *testing.T) {
	client, err := NewClient(Config{Address: "localhost:6379"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err := client.Enqueue(context.Background(), nil); err == nil {
		t.Fatal("expected nil task error")
	}
}

func TestNewServerDefaults(t *testing.T) {
	mux := asynq.NewServeMux()
	server, err := NewServer(ServerConfig{Redis: Config{Address: "localhost:6379"}}, mux)
	if err != nil {
		t.Fatal(err)
	}
	if server == nil {
		t.Fatal("expected server")
	}
}

func TestServerConfigShutdownTimeout(t *testing.T) {
	mux := asynq.NewServeMux()
	server, err := NewServer(ServerConfig{
		Redis: Config{Address: "localhost:6379"},
		ShutdownTimeout: time.Second,
	}, mux)
	if err != nil {
		t.Fatal(err)
	}
	if server == nil {
		t.Fatal("expected server")
	}
}
