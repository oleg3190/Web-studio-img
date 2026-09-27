package queue

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

var ErrInvalidRedisConfig = errors.New("invalid redis configuration")

type Config struct {
	Address  string
	Password string
	DB       int
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

func ParseRedisURL(value string) (Config, error) {
	value = strings.TrimSpace(value)
	if value == "" { return Config{}, ErrInvalidRedisConfig }
	if !strings.Contains(value, "://") { return Config{Address: value}, nil }
	u, err := url.Parse(value)
	if err != nil || u.Host == "" { return Config{}, fmt.Errorf("%w: invalid redis URL", ErrInvalidRedisConfig) }
	db := 0
	if strings.Trim(u.Path, "/") != "" {
		db, err = strconv.Atoi(strings.Trim(u.Path, "/")); if err != nil || db < 0 { return Config{}, fmt.Errorf("%w: invalid database", ErrInvalidRedisConfig) }
	}
	password, _ := u.User.Password()
	return Config{Address: u.Host, Password: password, DB: db}, nil
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
	})}, nil
}

func (c *Client) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if task == nil {
		return nil, errors.New("task is nil")
	}
	return c.inner.EnqueueContext(ctx, task, opts...)
}

func (c *Client) Ping(ctx context.Context) error {
	return c.inner.Ping()
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
