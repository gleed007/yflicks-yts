package yts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/atifcppprogrammer/yflicks-yts/internal/validate"
)

const APIBaseURL = "https://yts.mx/api/v2"

type Client struct {
	apiBaseURL    string
	apiHTTPClient *http.Client
}

type FilterValidationError validate.StructValidationError

func NewClient() *Client {
	return &Client{APIBaseURL, &http.Client{}}
}

func (c Client) SearchMovies(ctx context.Context, filters *SearchMoviesFilters) (*SearchMoviesResponse, error) {
	queryString, err := filters.getQueryString()
	if err != nil {
		return nil, err
	}

	parsedPayload := &SearchMoviesResponse{}
	targetURL := c.getEndpointURL("list_movies.json", queryString)
	err = c.getPayload(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c Client) GetMovieDetails(ctx context.Context, filters *MovieDetailsFilters) (*MovieDetailsResponse, error) {
	queryString, err := filters.getQueryString()
	if err != nil {
		return nil, err
	}

	parsedPayload := &MovieDetailsResponse{}
	targetURL := c.getEndpointURL("movie_details.json", queryString)
	err = c.getPayload(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c Client) GetMovieSuggestions(ctx context.Context, movieID int) (*MovieSuggestionsResponse, error) {
	if movieID <= 0 {
		return nil, errors.New("provided movieID must be at least 1")
	}

	var (
		movieIDStr  = fmt.Sprintf("%d", movieID)
		queryValues = url.Values{"movie_id": []string{movieIDStr}}
		queryString = queryValues.Encode()
	)

	parsedPayload := &MovieSuggestionsResponse{}
	targetURL := c.getEndpointURL("movie_suggestions.json", queryString)
	err := c.getPayload(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c Client) getPayload(ctx context.Context, targetURL string, payload interface{}) error {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	parsed := parsedURL.String()
	request, err := http.NewRequestWithContext(ctx, "GET", parsed, http.NoBody)
	if err != nil {
		return err
	}

	response, err := c.apiHTTPClient.Do(request)
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

func (c Client) getEndpointURL(path, query string) string {
	targetURL := fmt.Sprintf("%s/%s", c.apiBaseURL, path)
	if query == "" {
		return targetURL
	}

	return fmt.Sprintf("%s?%s", targetURL, query)
}
