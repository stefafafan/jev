package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stefafafan/jev/internal/evaluation"
)

type systemOneClient struct {
	httpClient *http.Client
	apiKey     string
	name       string
	model      string
	endpoint   string
}

func newSystemOneClient(httpClient *http.Client, apiKey, name, model, endpoint string) *systemOneClient {
	return &systemOneClient{httpClient: httpClient, apiKey: apiKey, name: name, model: model, endpoint: endpoint}
}

func (c *systemOneClient) Evaluate(ctx context.Context, request evaluation.Request) (evaluation.Response, error) {
	wireRequest := struct {
		State     string                         `json:"state"`
		Model     string                         `json:"model"`
		Questions map[string]evaluation.Question `json:"questions"`
	}{request.State, c.model, request.Questions}

	var response evaluation.Response
	if err := PostJSON(ctx, c.httpClient, c.endpoint, c.apiKey, wireRequest, &response); err != nil {
		return evaluation.Response{}, fmt.Errorf("%s evaluate request: %w", c.name, err)
	}
	response.Provider = c.name
	if err := evaluation.ValidateResponse(request, response); err != nil {
		return evaluation.Response{}, fmt.Errorf("%s validate response: %w", c.name, err)
	}
	return response, nil
}

var _ Provider = (*systemOneClient)(nil)
