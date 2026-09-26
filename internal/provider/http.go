package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const ResponseLimit = 4 << 20

func PostJSON(
	ctx context.Context,
	client *http.Client,
	endpoint, token string,
	requestBody, responseBody any,
) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode request: unsupported value")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("prepare request: invalid endpoint")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("request canceled: %w", ctxErr)
		}
		return fmt.Errorf("request failed: network error")
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("request failed: HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, ResponseLimit+1))
	if err != nil {
		return fmt.Errorf("read response: I/O error")
	}
	if len(data) > ResponseLimit {
		return fmt.Errorf("read response: body exceeds %d bytes", ResponseLimit)
	}
	if err := json.Unmarshal(data, responseBody); err != nil {
		return fmt.Errorf("decode response: invalid JSON")
	}
	return nil
}
