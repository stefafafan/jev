package main

import (
	"net/http"
	"testing"

	"github.com/stefafafan/jev/internal/provider"
)

func TestNewProvider(t *testing.T) {
	httpClient := &http.Client{}
	tests := []struct {
		name   string
		config provider.Config
	}{
		{"typesafe", provider.Config{Name: "typesafe", TypeSafeAPIKey: "token"}},
		{"cloudflare", provider.Config{Name: "cloudflare", CloudflareAPIToken: "token", CloudflareAccountID: "account"}},
		{"vercel", provider.Config{Name: "vercel", VercelAPIKey: "token"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := newProvider(httpClient, tt.config)
			if err != nil {
				t.Fatal(err)
			}
			if value == nil {
				t.Fatal("provider is nil")
			}
		})
	}
	if _, err := newProvider(httpClient, provider.Config{Name: "unknown"}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name     string
		injected string
		module   string
		want     string
	}{
		{"release archive", "0.1.0", "v0.1.0", "0.1.0"},
		{"go install", "", "v0.1.0", "0.1.0"},
		{"local build", "", "(devel)", "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.injected, tt.module); got != tt.want {
				t.Fatalf("resolveVersion(%q, %q) = %q, want %q", tt.injected, tt.module, got, tt.want)
			}
		})
	}
}
