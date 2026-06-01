package api

import (
	"context"
	"encoding/json"
	"errors"
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

func TestClientAcceptsCompleteJSONWithUnexpectedEOF(t *testing.T) {
	client := New("https://example.test", "", "server_mock")
	client.HTTPClient = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       errReadCloser{Reader: strings.NewReader(`{"InboundMessages":[],"TotalCount":0}`), err: io.ErrUnexpectedEOF},
		}, nil
	})

	raw, err := client.Get(context.Background(), ServerToken, "/messages/inbound", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got := string(raw); got != `{"InboundMessages":[],"TotalCount":0}` {
		t.Fatalf("raw = %q", got)
	}
}

func TestClientRejectsIncompleteJSONWithUnexpectedEOF(t *testing.T) {
	client := New("https://example.test", "", "server_mock")
	client.HTTPClient = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       errReadCloser{Reader: strings.NewReader(`{"InboundMessages":[`), err: io.ErrUnexpectedEOF},
		}, nil
	})

	_, err := client.Get(context.Background(), ServerToken, "/messages/inbound", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("error = %q", err.Error())
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

type errReadCloser struct {
	io.Reader
	err error
}

func (r errReadCloser) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == nil {
		return n, nil
	}
	if errors.Is(err, io.EOF) {
		return n, r.err
	}
	return n, err
}

func (r errReadCloser) Close() error {
	return nil
}
