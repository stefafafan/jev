package provider

import "net/http"

func NewVercel(httpClient *http.Client, apiKey string) Provider {
	return newSystemOneClient(httpClient, apiKey, "vercel", "typesafe-ai/jev", "https://ai-gateway.vercel.sh/typesafe/v1/systemone")
}
