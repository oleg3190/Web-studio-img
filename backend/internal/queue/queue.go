package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

var ErrInvalidRedisConfig = errors.New("invalid redis configuration")

type Config struct {
	Address  string
	Password string
	DB       int
	Prefix   string
}

func (c Config) validate() error {
	if c.Address == "" {
		return fmt.Errorf("%w: address is required", ErrInvalidRedisConfig)
	}
	if c.DB < 0 {
		return fmt.Errorf("%w: db must be non-negative", ErrInvalidRedisConfig)
	}
	return nil
}

type Client struct {
	inner *asynq.Client
}

func NewClient(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Client{inner: asynq.NewClient(asynq.RedisClientOpt{
		Addr:      cfg.Address,
		Password:  cfg.Password,
		DB:        cfg.DB,
		Namespace: cfg.Prefix,
	})}, nil
}

func (c *Client) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if task == nil {
		return nil, errors.New("task is nil")
	}
	return c.inner.EnqueueContext(ctx, task, opts...)
}

func (c *Client) Ping(ctx context.Context) error {
	return c.inner.PingContext(ctx)
}

func (c *Client) Close() error {
	return c.inner.Close()
}

type Server struct {
	inner *asynq.Server
}

type ServerConfig struct {
	Redis           Config
	Concurrency     int
	Queues          map[string]int
	ShutdownTimeout time.Duration
}

func NewServer(cfg ServerConfig, mux *asynq.ServeMux) (*Server, error) {
	if err := cfg.Redis.validate(); err != nil {
		return nil, err
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 30 * time.Second
	}
	if mux == nil {
		return nil, errors.New("serve mux is nil")
	}
	return &Server{inner: asynq.NewServer(asynq.RedisClientOpt{
		Addr:      cfg.Redis.Address,
		Password:  cfg.Redis.Password,
		DB:        cfg.Redis.DB,
		Namespace: cfg.Redis.Prefix,
	}, asynq.Config{
		Concurrency:     cfg.Concurrency,
		Queues:          cfg.Queues,
		ShutdownTimeout: cfg.ShutdownTimeout,
	})}, nil
}

func (s *Server) Run(mux *asynq.ServeMux) error {
	if mux == nil {
		return errors.New("serve mux is nil")
	}
	return s.inner.Run(mux)
}

func (s *Server) Shutdown() {
	s.inner.Shutdown()
}
