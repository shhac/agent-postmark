package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientUsesServerTokenHeader(t *testing.T) {
	client := New("https://example.test", "account_mock", "server_mock")
	client.HTTPClient = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("X-Postmark-Server-Token"); got != "server_mock" {
			t.Errorf("server token header = %q", got)
		}
		if got := r.Header.Get("X-Postmark-Account-Token"); got != "" {
			t.Errorf("account token header = %q", got)
		}
		return jsonResponse(http.StatusOK, map[string]any{"ok": true}), nil
	})

	raw, err := client.Get(context.Background(), ServerToken, "/messages/outbound", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("expected response body")
	}
}

func TestClientUsesAccountTokenHeader(t *testing.T) {
	client := New("https://example.test", "account_mock", "server_mock")
	client.HTTPClient = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("X-Postmark-Account-Token"); got != "account_mock" {
			t.Errorf("account token header = %q", got)
		}
		if got := r.Header.Get("X-Postmark-Server-Token"); got != "" {
			t.Errorf("server token header = %q", got)
		}
		return jsonResponse(http.StatusOK, map[string]any{"ok": true}), nil
	})

	if _, err := client.Get(context.Background(), AccountToken, "/servers", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

func TestClientMapsUnauthorized(t *testing.T) {
	client := New("https://example.test", "bad", "")
	client.HTTPClient = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, map[string]any{"ErrorCode": 10, "Message": "Bad or missing API token"}), nil
	})
	_, err := client.Get(context.Background(), AccountToken, "/servers", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), "Authentication failed: Bad or missing API token (ErrorCode 10)"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func jsonResponse(status int, body any) *http.Response {
	var b strings.Builder
	_ = json.NewEncoder(&b).Encode(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(b.String())),
	}
}
