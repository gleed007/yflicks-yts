package yts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const APIBaseURL = "https://yts.mx/api/v2"

type Client struct {
	apiBaseURL string
	apiClient  *http.Client
}

func NewClient() *Client {
	return &Client{APIBaseURL, &http.Client{}}
}

func (c Client) SearchMovies(filters *SearchMoviesFilters) (*SearchMoviesResponse, error) {
	queryString := filters.getQueryString()
	parsedPayload := &SearchMoviesResponse{}
	targetURL := c.getEndpointURL("list_movies.json", queryString)
	err := c.getEndpointPayload(targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c Client) getEndpointURL(path, query string) string {
	targetURL := fmt.Sprintf("%s/%s", c.apiBaseURL, path)
	if query == "" {
		return targetURL
	}

	return fmt.Sprintf("%s?%s", targetURL, query)
}

func (c Client) getEndpointPayload(targetURL string, payload interface{}) error {
	response, err := c.apiClient.Get(targetURL)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	rawPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(rawPayload, payload)
	if err != nil {
		return err
	}

	return nil
}
