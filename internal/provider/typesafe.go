package provider

import "net/http"

func NewTypeSafe(httpClient *http.Client, apiKey string) Provider {
	return newSystemOneClient(httpClient, apiKey, "typesafe", "jev-latest", "https://api.typesafe.ai/v1/systemone")
}
