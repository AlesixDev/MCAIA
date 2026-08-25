package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AlesixDev/MCAIA/backend/internal/ai"
	"github.com/AlesixDev/MCAIA/backend/internal/motion"
)

type Options struct {
	BaseURL     string
	Model       string
	Temperature float64
	NumCtx      int
	Think       bool
	Timeout     time.Duration
}

type Engine struct {
	options Options
	client  *http.Client
}

func New(options Options) *Engine {
	if options.BaseURL == "" {
		options.BaseURL = "http://127.0.0.1:11434"
	}

	if options.Model == "" {
		options.Model = "qwen3:14b"
	}

	if options.Timeout == 0 {
		options.Timeout = 3 * time.Minute
	}

	return &Engine{
		options: options,
		client:  &http.Client{Timeout: options.Timeout},
	}
}

func (e *Engine) ID() string {
	return "ollama:" + e.options.Model
}

func (e *Engine) Available(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, e.options.BaseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	response, err := e.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ai.ErrEngineUnavailable, err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ai.ErrEngineUnavailable, response.StatusCode)
	}

	return nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []message      `json:"messages"`
	Stream   bool           `json:"stream"`
	Think    bool           `json:"think"`
	Format   map[string]any `json:"format,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatResponse struct {
	Message message `json:"message"`
	Error   string  `json:"error,omitempty"`
}

func (e *Engine) Generate(ctx context.Context, request ai.Request) (*motion.Plan, error) {
	payload := chatRequest{
		Model:  e.options.Model,
		Stream: false,
		Think:  e.options.Think,
		Format: ai.ResponseSchema(),
		Messages: []message{
			{Role: "system", Content: ai.SystemPrompt()},
			{Role: "user", Content: ai.BuildPrompt(request)},
		},
		Options: map[string]any{
			"temperature": e.options.Temperature,
		},
	}

	if e.options.NumCtx > 0 {
		payload.Options["num_ctx"] = e.options.NumCtx
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, e.options.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := e.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ai.ErrEngineUnavailable, err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ai.ErrEngineUnavailable, response.StatusCode)
	}

	var decoded chatResponse

	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: %v", ai.ErrInvalidResponse, err)
	}

	if decoded.Error != "" {
		return nil, fmt.Errorf("%w: %s", ai.ErrEngineUnavailable, decoded.Error)
	}

	return ai.DecodePlan(decoded.Message.Content)
}
