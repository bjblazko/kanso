// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Errors a person can understand.
var (
	ErrUnreachable = errors.New("the AI cannot be reached")
	ErrAuth        = errors.New("the AI refused the token")
	ErrModel       = errors.New("the AI does not know the model")
	ErrTimeout     = errors.New("the AI took too long")
	ErrBadAnswer   = errors.New("the AI gave no usable answer")
)

const (
	// Timeout covers waking the machine, loading the model and answering.
	Timeout   = 180 * time.Second
	maxAnswer = 1 << 20
)

// Message is one turn of the conversation.
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// Client sends chat completions; the zero value uses its own HTTP client.
type Client struct {
	HTTP *http.Client
}

var thinking = regexp.MustCompile(`(?s)<think>.*?</think>`)

// Complete asks the model and returns its answer without any <think> part.
func (c Client) Complete(ctx context.Context, cfg Config, msgs []Message) (string, error) {
	body, _ := json.Marshal(map[string]any{"model": cfg.Model, "messages": msgs, "stream": false, "temperature": 0.2})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return "", sendError(ctx, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswer))
	if err != nil {
		return "", sendError(ctx, err)
	}
	if err := statusError(resp.StatusCode, b); err != nil {
		return "", err
	}
	return answer(b)
}

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: Timeout}
}

func sendError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	var ne interface{ Timeout() bool }
	if errors.As(err, &ne) && ne.Timeout() {
		return ErrTimeout
	}
	return fmt.Errorf("%w: %v", ErrUnreachable, err)
}

func statusError(status int, body []byte) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrAuth
	case status == http.StatusNotFound && bytes.Contains(bytes.ToLower(body), []byte("model")):
		return ErrModel
	case status >= 300:
		return fmt.Errorf("%w: HTTP %d", ErrUnreachable, status)
	}
	return nil
}

func answer(b []byte) (string, error) {
	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &out); err != nil || len(out.Choices) == 0 {
		return "", ErrBadAnswer
	}
	text := strings.TrimSpace(thinking.ReplaceAllString(out.Choices[0].Message.Content, ""))
	if text == "" {
		return "", ErrBadAnswer
	}
	return text, nil
}
