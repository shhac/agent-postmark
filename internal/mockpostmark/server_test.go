package mockpostmark

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockServersList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/servers?count=50&offset=0", nil)
	req.Header.Set("X-Postmark-Account-Token", "account_mock")
	rec := httptest.NewRecorder()

	NewServer().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		TotalCount int              `json:"TotalCount"`
		Servers    []map[string]any `json:"Servers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.TotalCount != 1 || len(payload.Servers) != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestMockRequiresToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/servers", nil)
	rec := httptest.NewRecorder()

	NewServer().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
