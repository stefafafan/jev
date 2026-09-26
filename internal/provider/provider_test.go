package provider

import (
	"strings"
	"testing"
)

func TestResolveProvider(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		env      map[string]string
		want     string
		wantErr  string
	}{
		{"explicit beats environment", "cloudflare", allCredentials("typesafe"), "cloudflare", ""},
		{"environment provider", "", allCredentials("cloudflare"), "cloudflare", ""},
		{"explicit Vercel", "vercel", map[string]string{"AI_GATEWAY_API_KEY": "vc-secret"}, "vercel", ""},
		{"environment Vercel", "", map[string]string{"JEV_PROVIDER": "vercel", "AI_GATEWAY_API_KEY": "vc-secret"}, "vercel", ""},
		{"auto TypeSafe AI", "", map[string]string{"TYPESAFE_API_KEY": "ts-secret"}, "typesafe", ""},
		{"auto Cloudflare", "", map[string]string{"CLOUDFLARE_API_TOKEN": "cf-secret", "CLOUDFLARE_ACCOUNT_ID": "account"}, "cloudflare", ""},
		{"auto Vercel", "", map[string]string{"AI_GATEWAY_API_KEY": "vc-secret"}, "vercel", ""},
		{"auto ignores partial other set", "", map[string]string{"TYPESAFE_API_KEY": "ts-secret", "CLOUDFLARE_API_TOKEN": "partial"}, "typesafe", ""},
		{"ambiguous", "", allCredentials(""), "", "multiple provider credentials"},
		{"ambiguous TypeSafe AI and Vercel", "", map[string]string{"TYPESAFE_API_KEY": "ts-secret", "AI_GATEWAY_API_KEY": "vc-secret"}, "", "multiple provider credentials"},
		{"partial Cloudflare", "cloudflare", map[string]string{"CLOUDFLARE_API_TOKEN": "cf-secret"}, "", "CLOUDFLARE_ACCOUNT_ID"},
		{"missing explicit credentials", "typesafe", nil, "", "TYPESAFE_API_KEY"},
		{"missing explicit Vercel credentials", "vercel", nil, "", "AI_GATEWAY_API_KEY"},
		{"empty values are absent", "", map[string]string{"TYPESAFE_API_KEY": "", "CLOUDFLARE_API_TOKEN": "", "CLOUDFLARE_ACCOUNT_ID": ""}, "", "provider is not configured"},
		{"unsupported case", "TypeSafe", map[string]string{"TYPESAFE_API_KEY": "ts-secret"}, "", "unsupported provider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := Resolve(tt.explicit, lookup(tt.env))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				for _, secret := range []string{"ts-secret", "cf-secret", "vc-secret", "partial"} {
					if strings.Contains(err.Error(), secret) {
						t.Fatalf("error leaked secret %q: %v", secret, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if config.Name != tt.want {
				t.Fatalf("name = %q, want %q", config.Name, tt.want)
			}
		})
	}
}

func TestResolvePreservesCredentialValues(t *testing.T) {
	env := map[string]string{
		"TYPESAFE_API_KEY":      "  token with spaces  ",
		"CLOUDFLARE_API_TOKEN":  "cf-token",
		"CLOUDFLARE_ACCOUNT_ID": "account/id",
	}
	config, err := Resolve("typesafe", lookup(env))
	if err != nil {
		t.Fatal(err)
	}
	if config.TypeSafeAPIKey != env["TYPESAFE_API_KEY"] {
		t.Fatalf("credential was changed: %q", config.TypeSafeAPIKey)
	}
}

func lookup(values map[string]string) LookupEnv {
	return func(key string) string { return values[key] }
}

func allCredentials(providerName string) map[string]string {
	return map[string]string{
		"JEV_PROVIDER":          providerName,
		"TYPESAFE_API_KEY":      "ts-secret",
		"CLOUDFLARE_API_TOKEN":  "cf-secret",
		"CLOUDFLARE_ACCOUNT_ID": "account",
		"AI_GATEWAY_API_KEY":    "vc-secret",
	}
}
