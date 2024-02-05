package yts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) newRequestWithContext(
	ctx context.Context, targetURL string,
) (*http.Response, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	parsed := parsedURL.String()
	request, err := http.NewRequestWithContext(ctx, "GET", parsed, http.NoBody)
	if err != nil {
		return nil, err
	}

	return c.netClient.Do(request)
}

func (c *Client) newJSONRequestWithContext(
	ctx context.Context, targetURL string, payload any,
) error {
	response, err := c.newRequestWithContext(ctx, targetURL)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	return decoder.Decode(payload)
}

func (c *Client) getAPIEndpoint(path, query string) string {
	targetURL := fmt.Sprintf("%s/%s", c.config.APIBaseURL, path)
	if query == "" {
		return targetURL
	}

	return fmt.Sprintf("%s?%s", targetURL, query)
}
