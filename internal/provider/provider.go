package provider

import (
	"context"
	"fmt"

	"github.com/stefafafan/jev/internal/evaluation"
)

type Provider interface {
	Evaluate(context.Context, evaluation.Request) (evaluation.Response, error)
}

type Config struct {
	Name                string
	TypeSafeAPIKey      string
	CloudflareAPIToken  string
	CloudflareAccountID string
	VercelAPIKey        string
}

type LookupEnv func(string) string

func Resolve(explicit string, getenv LookupEnv) (Config, error) {
	config := Config{
		TypeSafeAPIKey:      getenv("TYPESAFE_API_KEY"),
		CloudflareAPIToken:  getenv("CLOUDFLARE_API_TOKEN"),
		CloudflareAccountID: getenv("CLOUDFLARE_ACCOUNT_ID"),
		VercelAPIKey:        getenv("AI_GATEWAY_API_KEY"),
	}
	name := explicit
	if name == "" {
		name = getenv("JEV_PROVIDER")
	}
	if name != "" {
		config.Name = name
		if err := validateSelected(config); err != nil {
			return Config{}, err
		}
		return config, nil
	}

	typeSafeComplete := config.TypeSafeAPIKey != ""
	cloudflareComplete := config.CloudflareAPIToken != "" && config.CloudflareAccountID != ""
	vercelComplete := config.VercelAPIKey != ""
	completeCount := 0
	for _, complete := range []bool{typeSafeComplete, cloudflareComplete, vercelComplete} {
		if complete {
			completeCount++
		}
	}
	switch {
	case completeCount > 1:
		return Config{}, fmt.Errorf("multiple provider credentials are configured; set --provider or JEV_PROVIDER")
	case typeSafeComplete:
		config.Name = "typesafe"
		return config, nil
	case cloudflareComplete:
		config.Name = "cloudflare"
		return config, nil
	case vercelComplete:
		config.Name = "vercel"
		return config, nil
	case config.CloudflareAPIToken != "":
		return Config{}, fmt.Errorf("cloudflare credentials are incomplete: CLOUDFLARE_ACCOUNT_ID is required")
	case config.CloudflareAccountID != "":
		return Config{}, fmt.Errorf("cloudflare credentials are incomplete: CLOUDFLARE_API_TOKEN is required")
	default:
		return Config{}, fmt.Errorf("provider is not configured; set --provider or provider credentials")
	}
}

func validateSelected(config Config) error {
	switch config.Name {
	case "typesafe":
		if config.TypeSafeAPIKey == "" {
			return fmt.Errorf("TypeSafe AI authentication requires TYPESAFE_API_KEY")
		}
	case "cloudflare":
		if config.CloudflareAPIToken == "" {
			return fmt.Errorf("cloudflare authentication requires CLOUDFLARE_API_TOKEN")
		}
		if config.CloudflareAccountID == "" {
			return fmt.Errorf("cloudflare authentication requires CLOUDFLARE_ACCOUNT_ID")
		}
	case "vercel":
		if config.VercelAPIKey == "" {
			return fmt.Errorf("vercel authentication requires AI_GATEWAY_API_KEY")
		}
	default:
		return fmt.Errorf("unsupported provider %q; expected typesafe, cloudflare, or vercel", config.Name)
	}
	return nil
}
