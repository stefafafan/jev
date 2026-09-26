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

func TestEvaluateUsesSystemOneContract(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		model string
		new   func(*http.Client, string) Provider
	}{
		{"typesafe", "/v1/systemone", "jev-latest", NewTypeSafe},
		{"vercel", "/typesafe/v1/systemone", "typesafe-ai/jev", NewVercel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path || r.Header.Get("Authorization") != "Bearer token" {
					t.Errorf("path=%q authorization=%q", r.URL.Path, r.Header.Get("Authorization"))
				}
				var body struct {
					State     string                         `json:"state"`
					Model     string                         `json:"model"`
					Questions map[string]evaluation.Question `json:"questions"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.State != "secret state" || body.Model != tt.model {
					t.Errorf("request = %#v", body)
				}
				_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"result":{"type":"noul","noul":0.9}},"usage":{"input_tokens":10,"output_tokens":2}}`)
			}))
			defer server.Close()

			client := tt.new(server.Client(), "token").(*systemOneClient)
			client.endpoint = server.URL + tt.path
			response, err := client.Evaluate(context.Background(), providerNoulRequest())
			if err != nil {
				t.Fatal(err)
			}
			if response.Provider != tt.name {
				t.Fatalf("provider = %q", response.Provider)
			}
		})
	}
}

func TestSystemOneErrorsUseProviderContextWithoutSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "secret response")
	}))
	defer server.Close()

	for _, tt := range []struct {
		name string
		new  func(*http.Client, string) Provider
	}{
		{"typesafe", NewTypeSafe},
		{"vercel", NewVercel},
	} {
		client := tt.new(server.Client(), "secret-token").(*systemOneClient)
		client.endpoint = server.URL
		_, err := client.Evaluate(context.Background(), providerNoulRequest())
		if err == nil || !strings.Contains(err.Error(), tt.name+" evaluate") {
			t.Fatalf("%s error = %v", tt.name, err)
		}
		for _, secret := range []string{"secret-token", "secret state", "secret response", server.URL} {
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error leaked %q: %v", secret, err)
			}
		}
	}
}

func providerNoulRequest() evaluation.Request {
	questions, _ := evaluation.NewInlineNoul("Safe?")
	return evaluation.Request{State: "secret state", Questions: questions}
}
