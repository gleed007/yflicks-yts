package yts

import (
	"context"
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
