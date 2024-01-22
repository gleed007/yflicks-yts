package yts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func getMockBaseResponse() BaseResponse {
	return BaseResponse{
		Status:        "status",
		StatusMessage: "Query was successful",
		Meta: Meta{
			APIVersion:     2,
			ServerTime:     1704384528,
			ServerTimezone: "CET",
			ExecutionTime:  "0 ms",
		},
	}
}

func getMockSearchMoviesResponse() SearchMoviesResponse {
	return SearchMoviesResponse{
		BaseResponse: getMockBaseResponse(),
		Data: SearchMoviesData{
			MovieCount: 5,
			PageNumber: 1,
			Movies:     []Movie{},
		},
	}
}

func getMockMovieDetailsResponse(movieID int) MovieDetailsResponse {
	return MovieDetailsResponse{
		BaseResponse: getMockBaseResponse(),
		Data: MovieDetailsData{
			Movie: MovieDetails{
				MoviePartial: MoviePartial{
					ID:    movieID,
					Title: "Oppenheimer",
				},
			},
		},
	}
}

func getMockMovieSuggestionsResponse() MovieSuggestionsResponse {
	return MovieSuggestionsResponse{
		BaseResponse: getMockBaseResponse(),
		Data: MovieSuggestionsData{
			MovieCount: 4,
			Movies:     []Movie{},
		},
	}
}

func getTestHandlerFor(pattern string, payload interface{}) *http.ServeMux {
	handler := func(w http.ResponseWriter, r *http.Request) {
		serialized, _ := json.Marshal(payload)
		fmt.Fprintf(w, "%s", serialized)
	}

	serveMux := &http.ServeMux{}
	serveMux.HandleFunc(pattern, handler)
	return serveMux
}

func TestNewClient(t *testing.T) {
	t.Run("panics if provided timeout is not within correct range", func(t *testing.T) {
		defer func() {
			expected := errors.New("YTS client timeout must be between 5 and 300 seconds inclusive")
			received, ok := recover().(error)
			if !ok || received == nil || received.Error() != expected.Error() {
				t.Errorf("received error %v, expected %v", received, expected)
			}
		}()
		NewClient(0)
	})
}

func TestSearchMovies(t *testing.T) {
	t.Run("returns error if provided filters result in invalid querystring", func(t *testing.T) {
		client := NewClient(time.Minute * 5)
		filters := DefaultSearchMoviesFilter()
		expected := &StructValidationError{
			Struct:   "SearchMoviesFilters",
			Field:    "Limit",
			Tag:      "min",
			Value:    -1,
			Expected: "1",
		}
		filters.Limit = -1
		_, received := client.SearchMovies(context.TODO(), filters)
		if received == nil || received.Error() != expected.Error() {
			t.Errorf("received error %v, expected %v", received, expected)
		}
	})

	t.Run("returns parsed SearchMoviesResponse from list_movies.json endpoint", func(t *testing.T) {
		expected := getMockSearchMoviesResponse()
		handler := getTestHandlerFor("/list_movies.json", expected)
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, SiteURL, &http.Client{}}
		filters := DefaultSearchMoviesFilter()
		received, err := client.SearchMovies(context.TODO(), filters)
		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if received.Data.MovieCount != expected.Data.MovieCount {
			t.Errorf(
				"received data.movieCount %d, expected data.movieCount %d",
				received.Data.MovieCount,
				expected.Data.MovieCount,
			)
		}

		if received.Data.PageNumber != expected.Data.PageNumber {
			t.Errorf(
				"received data.pageNumber %d, expected data.pageNumber %d",
				received.Data.PageNumber,
				expected.Data.PageNumber,
			)
		}
	})
}

func TestGetMovieDetails(t *testing.T) {
	t.Run("returns error if provided filters result in invalid querystring", func(t *testing.T) {
		client := NewClient(time.Minute * 5)
		filters := DefaultMovieDetailsFilters(-1)
		expected := &StructValidationError{
			Struct:   "MovieDetailsFilters",
			Field:    "MovieID",
			Tag:      "min",
			Value:    -1,
			Expected: "1",
		}
		_, received := client.GetMovieDetails(context.TODO(), filters)
		if received == nil || received.Error() != expected.Error() {
			t.Errorf("received error %v, expected %v", received, expected)
		}
	})

	t.Run("returns parsed MovieDetailsResponse from movie_details.json endpoint", func(t *testing.T) {
		const movieID = 1
		expected := getMockMovieDetailsResponse(movieID)
		handler := getTestHandlerFor("/movie_details.json", expected)
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, SiteURL, &http.Client{}}
		filters := DefaultMovieDetailsFilters(movieID)
		received, err := client.GetMovieDetails(context.TODO(), filters)
		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if received.Data.Movie.ID != expected.Data.Movie.ID {
			t.Errorf(
				"received data.movie.id %d, expected data.movie.id %d",
				received.Data.Movie.ID,
				expected.Data.Movie.ID,
			)
		}

		if received.Data.Movie.Title != expected.Data.Movie.Title {
			t.Errorf(
				"received data.movie.title %s, expected data.movie.title %s",
				received.Data.Movie.Title,
				expected.Data.Movie.Title,
			)
		}
	})
}

func TestGetMovieSuggestions(t *testing.T) {
	t.Run("returns error if provided movieID results in invalid querystring", func(t *testing.T) {
		client := NewClient(time.Minute * 5)
		expected := errors.New("provided movieID must be at least 1")
		_, received := client.GetMovieSuggestions(context.TODO(), -1)
		if received == nil || received.Error() != expected.Error() {
			t.Errorf("received error %v, expected %v", received, expected)
		}
	})

	t.Run("returns parsed MovieSuggestionsResponse from movie_suggestions.json endpoint", func(t *testing.T) {
		const movieID = 1
		expected := getMockMovieSuggestionsResponse()
		handler := getTestHandlerFor("/movie_suggestions.json", expected)
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, SiteURL, &http.Client{}}
		received, err := client.GetMovieSuggestions(context.TODO(), movieID)
		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if received.Data.MovieCount != expected.Data.MovieCount {
			t.Errorf(
				"received data.movieCount %d, expected data.movieCount %d",
				received.Data.MovieCount,
				expected.Data.MovieCount,
			)
		}
	})
}

func TestGetPayloadJSON(t *testing.T) {
	client := NewClient(time.Minute * 5)
	t.Run("populates passed struct with response payload from server endpoint", func(t *testing.T) {
		expected := TestEmployee{"employee", 5000}
		handler := getTestHandlerFor("/", expected)
		server := httptest.NewServer(handler)
		defer server.Close()

		received := TestEmployee{}
		err := client.getPayloadJSON(context.TODO(), server.URL, &received)
		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if received.Name != expected.Name {
			t.Errorf("received name %s, expected %s", received.Name, expected.Name)
		}

		if received.Salary != expected.Salary {
			t.Errorf("received salary %d, expected %d", received.Salary, expected.Salary)
		}
	})
}

func TestGetPayloadRaw(t *testing.T) {
	client := NewClient(time.Minute * 5)

	t.Run("returns error if ill-formed URL provided as argument", func(t *testing.T) {
		malformedURL := "proto://malformed-url.com"
		received := client.getPayloadJSON(context.TODO(), malformedURL, struct{}{})
		expected := fmt.Errorf(`Get "%s": unsupported protocol scheme "proto"`, malformedURL)
		if received == nil || received.Error() != expected.Error() {
			t.Errorf("received error %s, expected %s", received, expected)
		}
	})

	t.Run("returns raw response from server as bytes", func(t *testing.T) {
		payload := TestEmployee{"employee", 5000}
		handler := getTestHandlerFor("/", payload)
		server := httptest.NewServer(handler)
		defer server.Close()

		rawPayload, err := client.getPayloadRaw(context.TODO(), server.URL)
		expected := `{"name":"employee","salary":5000}`
		received := string(rawPayload)
		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if received != expected {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})
}

func TestGetEndpointURL(t *testing.T) {
	const targetPath = "list_movies.json"
	client := NewClient(time.Minute * 5)

	t.Run("generates correct target URL when empty querystring is provided", func(t *testing.T) {
		received := client.getEndpointURL(targetPath, "")
		expected := fmt.Sprintf("%s/%s", APIBaseURL, targetPath)
		if received != expected {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})

	t.Run("generates correct target URL when non-empty querystring is provided", func(t *testing.T) {
		queryString, _ := DefaultSearchMoviesFilter().getQueryString()
		received := client.getEndpointURL(targetPath, queryString)
		expected := fmt.Sprintf("%s/%s?%s", APIBaseURL, targetPath, queryString)
		if received != expected {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})
}
