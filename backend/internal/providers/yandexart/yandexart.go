package yandexart

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oleg3190/Web-studio-img/backend/internal/generation"
)

var (
	ErrProviderUnavailable = errors.New("yandexart provider unavailable")
	ErrProviderInvalid     = errors.New("yandexart request rejected")
	ErrProviderCancelled   = errors.New("yandexart operation cancelled")
)

type Config struct {
	Endpoint  string
	OperationEndpoint string
	APIKey    string
	FolderID  string
	Model     string
	Timeout   time.Duration
	PollEvery time.Duration
}

type Provider struct {
	cfg    Config
	client *http.Client
}

type retryableError struct{ err error }

func (e retryableError) Error() string { return e.err.Error() }
func (e retryableError) Unwrap() error { return e.err }
func (e retryableError) Retryable() bool { return true }

func New(cfg Config) (*Provider, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://llm.api.cloud.yandex.net"
	}
	if cfg.OperationEndpoint == "" {
		cfg.OperationEndpoint = "https://operation.api.cloud.yandex.net"
	}
	if cfg.Model == "" {
		cfg.Model = "yandex-art/latest"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Minute
	}
	if cfg.PollEvery <= 0 {
		cfg.PollEvery = 2 * time.Second
	}
	if cfg.APIKey == "" || cfg.FolderID == "" {
		return nil, errors.New("yandexart requires API key and folder id")
	}
	return &Provider{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (p *Provider) Name() string { return "yandexart" }
func (p *Provider) Model() string { return p.cfg.Model }

type generateRequest struct {
	ModelURI        string `json:"modelUri"`
	Messages        []message `json:"messages"`
	GenerationOpts generationOptions `json:"generationOptions"`
}
type message struct {
	Text   string `json:"text"`
	Weight string `json:"weight,omitempty"`
}
type generationOptions struct {
	MIMEType     string `json:"mimeType"`
	Seed         string `json:"seed,omitempty"`
	AspectRatio  *aspectRatio `json:"aspectRatio,omitempty"`
}
type aspectRatio struct {
	WidthRatio  string `json:"widthRatio"`
	HeightRatio string `json:"heightRatio"`
}
type operation struct {
	ID       string `json:"id"`
	Done     bool `json:"done"`
	Error    *operationError `json:"error,omitempty"`
	Response *imageResponse `json:"response,omitempty"`
}
type operationError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type imageResponse struct {
	Image        string `json:"image"`
	ModelVersion string `json:"modelVersion"`
}

func (p *Provider) Generate(ctx context.Context, req generation.Request) (generation.ProviderResult, error) {
	positive := strings.TrimSpace(req.Prompt)
	payload := generateRequest{
		ModelURI: "art://" + p.cfg.FolderID + "/" + p.cfg.Model,
		Messages: []message{{Text: positive, Weight: "1"}},
		GenerationOpts: generationOptions{MIMEType: "image/jpeg"},
	}
	if strings.TrimSpace(req.NegativePrompt) != "" {
		payload.Messages = append(payload.Messages, message{Text: strings.TrimSpace(req.NegativePrompt), Weight: "-1"})
	}
	if req.Seed != nil {
		payload.GenerationOpts.Seed = strconv.FormatInt(*req.Seed, 10)
	}
	if w, h := parseAspectRatio(req.AspectRatio); w != "" {
		payload.GenerationOpts.AspectRatio = &aspectRatio{WidthRatio: w, HeightRatio: h}
	}
	var op operation
	if err := p.doJSON(ctx, http.MethodPost, "/foundationModels/v1/imageGenerationAsync", payload, &op); err != nil {
		return generation.ProviderResult{}, err
	}
	if op.ID == "" {
		return generation.ProviderResult{}, fmt.Errorf("%w: missing operation id", ErrProviderUnavailable)
	}

	pollCtx, cancel := context.WithTimeout(ctx, p.cfg.Timeout)
	defer cancel()
	for {
		if err := pollCtx.Err(); err != nil {
			return generation.ProviderResult{}, err
		}
		if op.Done {
			break
		}
		timer := time.NewTimer(p.cfg.PollEvery)
		select {
		case <-pollCtx.Done():
			timer.Stop()
			return generation.ProviderResult{}, pollCtx.Err()
		case <-timer.C:
		}
		if err := p.doJSONURL(pollCtx, p.cfg.OperationEndpoint, http.MethodGet, "/operations/"+op.ID, nil, &op); err != nil {
			return generation.ProviderResult{}, err
		}
	}
	if op.Error != nil {
		if op.Error.Code == 1 || op.Error.Code == 4 || op.Error.Code == 14 {
			return generation.ProviderResult{}, fmt.Errorf("%w: %s", ErrProviderUnavailable, op.Error.Message)
		}
		return generation.ProviderResult{}, fmt.Errorf("%w: %s", ErrProviderInvalid, op.Error.Message)
	}
	if op.Response == nil || op.Response.Image == "" {
		return generation.ProviderResult{}, fmt.Errorf("%w: empty image response", ErrProviderUnavailable)
	}
	data, err := base64.StdEncoding.DecodeString(op.Response.Image)
	if err != nil {
		return generation.ProviderResult{}, fmt.Errorf("decode yandexart image: %w", err)
	}
	return generation.ProviderResult{
		Images: []generation.Image{{Data: data, ContentType: "image/jpeg", Filename: "generation.jpg"}},
		ModelVersion: op.Response.ModelVersion,
		ProviderJobID: op.ID,
	}, nil
}

func (p *Provider) Cancel(ctx context.Context, operationID string) error {
	if operationID == "" {
		return nil
	}
	var op operation
	if err := p.doJSONURL(ctx, p.cfg.OperationEndpoint, http.MethodGet, "/operations/"+operationID+":cancel", nil, &op); err != nil {
		return err
	}
	return nil
}

func (p *Provider) doJSON(ctx context.Context, method, path string, body, out any) error {
	return p.doJSONURL(ctx, p.cfg.Endpoint, method, path, body, out)
}

func (p *Provider) doJSONURL(ctx context.Context, endpoint, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode yandexart request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Api-Key "+p.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return retryableError{fmt.Errorf("%w: %v", ErrProviderUnavailable, err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
			return fmt.Errorf("%w: status=%d body=%s", ErrProviderInvalid, resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return retryableError{fmt.Errorf("%w: status=%d body=%s", ErrProviderUnavailable, resp.StatusCode, strings.TrimSpace(string(body)))}
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode yandexart response: %w", err)
	}
	return nil
}

func parseAspectRatio(value string) (string, string) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return "", ""
	}
	w, h := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if _, err := strconv.Atoi(w); err != nil {
		return "", ""
	}
	if _, err := strconv.Atoi(h); err != nil {
		return "", ""
	}
	return w, h
}
