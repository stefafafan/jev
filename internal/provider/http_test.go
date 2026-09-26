package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostJSONSendsRequestAndDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"state":"secret-state"}` {
			t.Errorf("body = %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	var response struct {
		OK bool `json:"ok"`
	}
	err := PostJSON(context.Background(), server.Client(), server.URL, "test-token", map[string]string{"state": "secret-state"}, &response)
	if err != nil {
		t.Fatal(err)
	}
	if !response.OK {
		t.Fatal("response was not decoded")
	}
}

func TestPostJSONRejectsUnsafeOrInvalidResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   func() string
	}{
		{"non-2xx", http.StatusUnauthorized, func() string { return `{"error":"secret-response"}` }},
		{"invalid JSON", http.StatusOK, func() string { return `not-json secret-response` }},
		{"too large", http.StatusOK, func() string { return strings.Repeat("x", ResponseLimit+1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body())
			}))
			defer server.Close()

			var response any
			err := PostJSON(context.Background(), server.Client(), server.URL+"/secret-url", "secret-token", map[string]string{"state": "secret-state"}, &response)
			if err == nil {
				t.Fatal("expected error")
			}
			for _, secret := range []string{"secret-token", "secret-state", "secret-response", "secret-url", server.URL} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("error leaked %q: %v", secret, err)
				}
			}
		})
	}
}

func TestPostJSONHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Context().Err() != context.Canceled {
			t.Fatalf("request context error = %v, want canceled", request.Context().Err())
		}
		return nil, context.Canceled
	})}
	err := PostJSON(ctx, client, "https://example.invalid/private", "secret-token", struct{}{}, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error = %v, want cancellation", err)
	}
}

func TestPostJSONClosesResponseBody(t *testing.T) {
	body := &trackingBody{Reader: bytes.NewBufferString(`{"ok":true}`)}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       body,
			Header:     make(http.Header),
		}, nil
	})}
	var response any
	if err := PostJSON(context.Background(), client, "https://example.invalid", "token", struct{}{}, &response); err != nil {
		t.Fatal(err)
	}
	if !body.closed {
		t.Fatal("response body was not closed")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}
