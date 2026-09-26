package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stefafafan/jev/internal/evaluation"
)

func TestEvaluateUsesCloudflareUnifiedContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/client/v4/accounts/acct%20123/ai/run" {
			t.Errorf("path = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer cf-token" {
			t.Errorf("Authorization = %q", got)
		}
		var body struct {
			Model string `json:"model"`
			Input struct {
				State     string                         `json:"state"`
				Questions map[string]evaluation.Question `json:"questions"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "typesafe/jev" || body.Input.State != "secret state" {
			t.Errorf("request = %#v", body)
		}
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"result":{"type":"noul","noul":0.8}},"usage":{"input_tokens":8,"output_tokens":2}}`)
	}))
	defer server.Close()

	client := NewCloudflare(server.Client(), "cf-token", "acct 123").(*cloudflareClient)
	client.baseURL = server.URL + "/client/v4/accounts"
	response, err := client.Evaluate(context.Background(), providerNoulRequest())
	if err != nil {
		t.Fatal(err)
	}
	if response.Provider != "cloudflare" || response.Model != "jev-1.13.0" {
		t.Fatalf("response = %#v", response)
	}
}

func TestEvaluateAcceptsCloudflareEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"result":{"state":"complete","result":{"model":"jev-1.13.0","answers":{"result":{"type":"noul","noul":0.8}},"usage":{"input_tokens":8,"output_tokens":2}},"gatewayMetadata":{"keySource":"cloudflare"}},"success":true}`)
	}))
	defer server.Close()
	client := NewCloudflare(server.Client(), "token", "account").(*cloudflareClient)
	client.baseURL = server.URL
	response, err := client.Evaluate(context.Background(), providerNoulRequest())
	if err != nil {
		t.Fatal(err)
	}
	if response.Provider != "cloudflare" || response.Model != "jev-1.13.0" {
		t.Fatalf("response = %#v", response)
	}
}

func TestEvaluateErrorsHaveSafeCloudflareContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, "secret response")
	}))
	defer server.Close()
	client := NewCloudflare(server.Client(), "secret-token", "secret-account").(*cloudflareClient)
	client.baseURL = server.URL
	_, err := client.Evaluate(context.Background(), providerNoulRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"cloudflare", "evaluate"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err, want)
		}
	}
	for _, secret := range []string{"secret-token", "secret-account", "secret state", "secret response", server.URL} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q: %v", secret, err)
		}
	}
}
