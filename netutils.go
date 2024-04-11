package yts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

var ErrUnexpectedHTTPResponseStatus = errors.New(
	"unexpected_http_response_status",
)

func (c *Client) newRequestWithContext(ctx context.Context, targetURL *url.URL) (
	*http.Response, error,
) {
	targetURLString := targetURL.String()
	request, err := http.NewRequestWithContext(ctx, "GET", targetURLString, http.NoBody)
	if err != nil {
		return nil, err
	}

	response, err := c.netClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || 299 < response.StatusCode {
		sErr := fmt.Errorf("received response with status code: %d", response.StatusCode)
		return nil, wrapErr(ErrUnexpectedHTTPResponseStatus, sErr)
	}

	return response, err
}

func (c *Client) newJSONRequestWithContext(
	ctx context.Context, targetURL *url.URL, payload any,
) error {
	response, err := c.newRequestWithContext(ctx, targetURL)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	return decoder.Decode(payload)
}

func (c *Client) newDocumentRequestWithContext(
	ctx context.Context, targetURL *url.URL,
) (*goquery.Document, error) {
	response, err := c.newRequestWithContext(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		debug.Println(err)
		return nil, ErrContentRetrievalFailure
	}

	return document, nil
}

func (c *Client) getAPIEndpoint(path, query string) string {
	targetURL := fmt.Sprintf("%s/%s", &c.config.APIBaseURL, path)
	if query == "" {
		return targetURL
	}

	return fmt.Sprintf("%s?%s", targetURL, query)
}
