package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/stefafafan/jev/internal/cli"
	"github.com/stefafafan/jev/internal/provider"
)

var version string

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpClient := &http.Client{Timeout: 60 * time.Second}
	app := cli.App{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getenv: os.Getenv,
		NewProvider: func(config provider.Config) (provider.Provider, error) {
			return newProvider(httpClient, config)
		},
		Version: buildVersion(),
	}
	os.Exit(app.Run(ctx, os.Args[1:]))
}

func buildVersion() string {
	moduleVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		moduleVersion = info.Main.Version
	}
	return resolveVersion(version, moduleVersion)
}

func resolveVersion(injected, module string) string {
	if injected != "" {
		return injected
	}
	if module == "" || module == "(devel)" {
		return "dev"
	}
	return strings.TrimPrefix(module, "v")
}

func newProvider(httpClient *http.Client, config provider.Config) (provider.Provider, error) {
	switch config.Name {
	case "typesafe":
		return provider.NewTypeSafe(httpClient, config.TypeSafeAPIKey), nil
	case "cloudflare":
		return provider.NewCloudflare(httpClient, config.CloudflareAPIToken, config.CloudflareAccountID), nil
	case "vercel":
		return provider.NewVercel(httpClient, config.VercelAPIKey), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", config.Name)
	}
}
