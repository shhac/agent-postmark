package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	agenterrors "github.com/shhac/agent-postmark/internal/errors"
)

type TokenKind string

const (
	AccountToken TokenKind = "account"
	ServerToken  TokenKind = "server"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	BaseURL      string
	AccountToken string
	ServerToken  string
	HTTPClient   Doer
	MaxRetries   int
	Debug        bool
	DebugOut     io.Writer
	Sleep        func(time.Duration)
}

func New(baseURL, accountToken, serverToken string) *Client {
	return &Client{
		BaseURL:      strings.TrimRight(baseURL, "/"),
		AccountToken: accountToken,
		ServerToken:  serverToken,
		HTTPClient:   http.DefaultClient,
		MaxRetries:   2,
		Sleep:        time.Sleep,
	}
}

func (c *Client) Get(ctx context.Context, kind TokenKind, path string, query url.Values) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, kind, path, query, nil)
}

func (c *Client) Post(ctx context.Context, kind TokenKind, path string, query url.Values, body any) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, kind, path, query, body)
}

func (c *Client) Put(ctx context.Context, kind TokenKind, path string, query url.Values, body any) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPut, kind, path, query, body)
}

func (c *Client) Delete(ctx context.Context, kind TokenKind, path string, query url.Values, body any) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, kind, path, query, body)
}

func (c *Client) do(ctx context.Context, method string, kind TokenKind, path string, query url.Values, body any) (json.RawMessage, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Sleep == nil {
		c.Sleep = time.Sleep
	}

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
		}
	}

	attempts := c.MaxRetries + 1
	for attempt := range attempts {
		req, err := c.newRequest(ctx, method, kind, path, query, payload)
		if err != nil {
			return nil, err
		}
		if c.Debug && c.DebugOut != nil {
			_ = json.NewEncoder(c.DebugOut).Encode(map[string]any{
				"debug":      "request",
				"method":     method,
				"url":        req.URL.String(),
				"token_kind": kind,
			})
		}
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			if attempt < attempts-1 {
				c.Sleep(backoff(attempt, 0))
				continue
			}
			return nil, agenterrors.Wrap(err, agenterrors.FixableByRetry).WithHint("Network request failed after retries.")
		}
		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, agenterrors.Wrap(readErr, agenterrors.FixableByRetry)
		}
		if c.Debug && c.DebugOut != nil {
			_ = json.NewEncoder(c.DebugOut).Encode(map[string]any{
				"debug":  "response",
				"status": resp.StatusCode,
				"url":    req.URL.String(),
			})
		}
		if retryable(resp.StatusCode) && attempt < attempts-1 {
			c.Sleep(backoff(attempt, retryAfter(resp.Header.Get("Retry-After"))))
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, mapError(resp.StatusCode, raw, kind)
		}
		if len(raw) == 0 {
			return json.RawMessage(`{}`), nil
		}
		return json.RawMessage(raw), nil
	}
	return nil, agenterrors.New("request failed after retries", agenterrors.FixableByRetry)
}

func (c *Client) newRequest(ctx context.Context, method string, kind TokenKind, path string, query url.Values, payload []byte) (*http.Request, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent).WithHint("Check the Postmark host or AGENT_POSTMARK_BASE_URL.")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	q := u.Query()
	for key, values := range query {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	u.RawQuery = q.Encode()

	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	switch kind {
	case AccountToken:
		if c.AccountToken == "" {
			return nil, agenterrors.New("missing Postmark account token", agenterrors.FixableByHuman).
				WithHint("Run 'agent-postmark profiles add <profile> --form --account-token' or use a server-token command.")
		}
		req.Header.Set("X-Postmark-Account-Token", c.AccountToken)
	case ServerToken:
		if c.ServerToken == "" {
			return nil, agenterrors.New("missing Postmark server token", agenterrors.FixableByHuman).
				WithHint("Run 'agent-postmark profiles add <profile> --form --server-token' or choose an account-token command.")
		}
		req.Header.Set("X-Postmark-Server-Token", c.ServerToken)
	default:
		return nil, agenterrors.Newf(agenterrors.FixableByAgent, "unknown token kind %q", kind)
	}
	return req, nil
}

func mapError(status int, raw []byte, kind TokenKind) error {
	detail := strings.TrimSpace(string(raw))
	var payload struct {
		ErrorCode int      `json:"ErrorCode"`
		Message   string   `json:"Message"`
		Errors    []string `json:"Errors"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil && payload.Message != "" {
		detail = payload.Message
		if payload.ErrorCode != 0 {
			detail = fmt.Sprintf("%s (ErrorCode %d)", detail, payload.ErrorCode)
		}
	}
	if detail == "" {
		detail = http.StatusText(status)
	}

	switch {
	case status == http.StatusUnauthorized:
		return agenterrors.New("Authentication failed: "+detail, agenterrors.FixableByHuman).
			WithHint(fmt.Sprintf("This endpoint uses the Postmark %s token header. Check the profile with 'agent-postmark auth check <profile>'.", kind))
	case status == http.StatusForbidden:
		return agenterrors.New("Permission denied: "+detail, agenterrors.FixableByHuman).
			WithHint("The token may not have access to this Postmark server or account resource.")
	case status == http.StatusNotFound:
		return agenterrors.New("Not found: "+detail, agenterrors.FixableByAgent).
			WithHint("Check the resource ID, default server, and message stream.")
	case status == http.StatusTooManyRequests:
		return agenterrors.New("Rate limited: "+detail, agenterrors.FixableByRetry).
			WithHint("Wait and retry, or reduce --count/page size.")
	case status >= 500:
		return agenterrors.New(fmt.Sprintf("Postmark server error (%d): %s", status, detail), agenterrors.FixableByRetry)
	default:
		return agenterrors.New(detail, agenterrors.FixableByAgent)
	}
}

func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func retryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		return time.Until(when)
	}
	return 0
}

func backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	ms := int(math.Pow(2, float64(attempt))) * 250
	return time.Duration(ms) * time.Millisecond
}
