package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/stefafafan/jev/internal/evaluation"
)

const cloudflareBaseURL = "https://api.cloudflare.com/client/v4/accounts"

type cloudflareClient struct {
	httpClient *http.Client
	apiToken   string
	accountID  string
	baseURL    string
}

func NewCloudflare(httpClient *http.Client, apiToken, accountID string) Provider {
	return &cloudflareClient{
		httpClient: httpClient,
		apiToken:   apiToken,
		accountID:  accountID,
		baseURL:    cloudflareBaseURL,
	}
}

func (c *cloudflareClient) Evaluate(ctx context.Context, request evaluation.Request) (evaluation.Response, error) {
	wireRequest := struct {
		Model string             `json:"model"`
		Input evaluation.Request `json:"input"`
	}{Model: "typesafe/jev", Input: request}
	endpoint := c.baseURL + "/" + url.PathEscape(c.accountID) + "/ai/run"
	var wireResponse struct {
		evaluation.Response
		Result *struct {
			Result evaluation.Response `json:"result"`
		} `json:"result"`
	}
	if err := PostJSON(ctx, c.httpClient, endpoint, c.apiToken, wireRequest, &wireResponse); err != nil {
		return evaluation.Response{}, fmt.Errorf("cloudflare evaluate request: %w", err)
	}
	response := wireResponse.Response
	if wireResponse.Result != nil {
		response = wireResponse.Result.Result
	}
	response.Provider = "cloudflare"
	if err := evaluation.ValidateResponse(request, response); err != nil {
		return evaluation.Response{}, fmt.Errorf("cloudflare validate response: %w", err)
	}
	return response, nil
}

var _ Provider = (*cloudflareClient)(nil)
