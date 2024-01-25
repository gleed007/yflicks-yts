package yts

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

//go:embed testdata/*
var testdata embed.FS

func getMockTrendingMoviesResponse() string {
	content, _ := testdata.ReadFile("testdata/page_trending.html")
	fmt.Println(content)
	return string(content)
}

func getMockHomePageContentResponse() string {
	content, _ := testdata.ReadFile("testdata/page_home.html")
	return string(content)
}

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

func getTestHandler(pattern string, payload []byte) *http.ServeMux {
	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", string(payload))
	}

	serveMux := &http.ServeMux{}
	serveMux.HandleFunc(pattern, handler)
	return serveMux
}

func getTestHandlerJSON(pattern string, payload interface{}) *http.ServeMux {
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
		handler := getTestHandlerJSON("/list_movies.json", expected)
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
		handler := getTestHandlerJSON("/movie_details.json", expected)
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
		handler := getTestHandlerJSON("/movie_suggestions.json", expected)
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

func TestGetTrendingMovies(t *testing.T) {
	t.Run("returns error if movie containing element is not found in document", func(t *testing.T) {
		payload := `<div>error-document</div>`
		handler := getTestHandler("/trending-movies", []byte(payload))
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, server.URL, &http.Client{}}
		_, received := client.GetTrendingMovies(context.TODO())
		expected := errors.New("no selections found for trending movies")
		if received == nil || received.Error() != expected.Error() {
			t.Errorf(`received error %v, but expected error "%v"`, received, expected)
		}
	})

	t.Run("returns parsed TrendingMoviesResponse from scraping YTS trending page", func(t *testing.T) {
		payload := getMockTrendingMoviesResponse()
		handler := getTestHandler("/trending-movies", []byte(payload))
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, server.URL, &http.Client{}}
		received, err := client.GetTrendingMovies(context.TODO())
		expected := TrendingMoviesResponse{
			Data: TrendingMoviesData{
				Movies: []ScrapedMovie{{
					Title:  "Superbad",
					Year:   2007,
					Link:   "https://yts.mx/movies/superbad-2007",
					Image:  "/assets/images/movies/Superbad_2007/medium-cover.jpg",
					Rating: "7.6 / 10",
				}},
			},
		}

		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if len(received.Data.Movies) != 1 {
			t.Errorf(
				"received movie count %d, expected %d",
				len(received.Data.Movies),
				1,
			)
		}

		if received.Data.Movies[0].Title != expected.Data.Movies[0].Title {
			t.Errorf(
				"received title %s, expected title %s",
				received.Data.Movies[0].Title,
				expected.Data.Movies[0].Title,
			)
		}

		if received.Data.Movies[0].Rating != expected.Data.Movies[0].Rating {
			t.Errorf(
				"received rating %s, expected rating %s",
				received.Data.Movies[0].Rating,
				expected.Data.Movies[0].Rating,
			)
		}
	})
}

func TestGetHomePageContent(t *testing.T) {
	t.Run("returns error if no popular movie containing element is found in document", func(t *testing.T) {
		payload := `<div>error-document</div>`
		handler := getTestHandler("/", []byte(payload))
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, server.URL, &http.Client{}}
		_, received := client.GetHomePageContent(context.TODO())
		expected := errors.New("no elements found for popular movies selection")
		if received == nil || received.Error() != expected.Error() {
			t.Errorf(`received error %v, but expected error "%v"`, received, expected)
		}
	})

	t.Run("returns error if no latest movie torrent containing element is found in document", func(t *testing.T) {
		payload, _ := testdata.ReadFile("testdata/latest_torrents_missing.html")
		handler := getTestHandler("/", payload)
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, server.URL, &http.Client{}}
		_, received := client.GetHomePageContent(context.TODO())
		expected := errors.New("no elements found for latest torrents selection")
		if received == nil || received.Error() != expected.Error() {
			t.Errorf(`received error %v, but expected error "%v"`, received, expected)
		}
	})

	t.Run("returns error if no upcoming movies containing element is found in document", func(t *testing.T) {
		payload, _ := testdata.ReadFile("testdata/upcoming_movies_missing.html")
		handler := getTestHandler("/", payload)
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, server.URL, &http.Client{}}
		_, received := client.GetHomePageContent(context.TODO())
		expected := errors.New("no elements found for upcoming movies selection")
		if received == nil || received.Error() != expected.Error() {
			t.Errorf(`received error %v, but expected error "%v"`, received, expected)
		}
	})

	t.Run("returns parsed HomePageContentResponse pfrom scraping YTS home page", func(t *testing.T) {
		payload := getMockHomePageContentResponse()
		handler := getTestHandler("/", []byte(payload))
		server := httptest.NewServer(handler)
		defer server.Close()

		client := Client{server.URL, server.URL, &http.Client{}}
		received, err := client.GetHomePageContent(context.TODO())
		expected := HomePageContentResponse{
			Data: HomePageContentData{
				Popular: []ScrapedMovie{{
					Title:  "Migration",
					Year:   2023,
					Link:   "https://yts.mx/movies/migration-2023",
					Image:  "/assets/images/movies/migration_2023/medium-cover.jpg",
					Rating: "6.8 / 10",
				}},
				Latest: []ScrapedMovie{{
					Title:  "[NL] Het einde van de reis",
					Year:   1981,
					Link:   "https://yts.mx/movies/het-einde-van-de-reis-1981",
					Image:  "/assets/images/movies/het_einde_van_de_reis_1981/medium-cover.jpg",
					Rating: "5.3 / 10",
				}},
				Upcoming: []ScrapedUpcomingMovie{{
					Progress: 28,
					ScrapedMovie: ScrapedMovie{
						Title: "Boyz n the Hood",
						Year:  1991,
						Link:  "https://www.imdb.com/title/tt0101507/",
						Image: "/assets/images/movies/Boyz_n_the_Hood_1991/medium-cover.jpg",
					},
				}},
			},
		}

		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if len(received.Data.Popular) != 1 {
			t.Errorf(
				"received popular movie count %d, expected %d",
				len(received.Data.Popular),
				1,
			)
		}

		if received.Data.Popular[0].Title != expected.Data.Popular[0].Title {
			t.Errorf(
				"received popular movie title %s, expected popular movie title %s",
				received.Data.Popular[0].Title,
				expected.Data.Popular[0].Title,
			)
		}

		if received.Data.Popular[0].Rating != expected.Data.Popular[0].Rating {
			t.Errorf(
				"received popular movie rating %s, expected popular movie rating %s",
				received.Data.Popular[0].Rating,
				expected.Data.Popular[0].Rating,
			)
		}

		if len(received.Data.Latest) != 1 {
			t.Errorf(
				"received latest torrents movie count %d, expected %d",
				len(received.Data.Latest),
				1,
			)
		}

		if received.Data.Latest[0].Title != expected.Data.Latest[0].Title {
			t.Errorf(
				"received latest torrent movie title  %s, expected latest torrent movie title %s",
				received.Data.Latest[0].Title,
				expected.Data.Latest[0].Title,
			)
		}

		if received.Data.Latest[0].Image != expected.Data.Latest[0].Image {
			t.Errorf(
				"received latest torrent movie image %s, expected latest torrent movie image %s",
				received.Data.Latest[0].Image,
				expected.Data.Latest[0].Image,
			)
		}

		if received.Data.Latest[0].Rating != expected.Data.Latest[0].Rating {
			t.Errorf(
				"received latest torrent movie rating %s, expected latest torrent movie rating %s",
				received.Data.Latest[0].Rating,
				expected.Data.Latest[0].Rating,
			)
		}

		if len(received.Data.Upcoming) != 1 {
			t.Errorf(
				"received upcoming movie count %d, expected %d",
				len(received.Data.Upcoming),
				1,
			)
		}

		if received.Data.Upcoming[0].Link != expected.Data.Upcoming[0].Link {
			t.Errorf(
				"received upcoming movie link %s, expected upcoming movie link %s",
				received.Data.Upcoming[0].Link,
				expected.Data.Upcoming[0].Link,
			)
		}

		if received.Data.Upcoming[0].Progress != expected.Data.Upcoming[0].Progress {
			t.Errorf(
				"received upcoming movie progress %d, expected upcoming movie progress %d",
				received.Data.Upcoming[0].Progress,
				expected.Data.Upcoming[0].Progress,
			)
		}
	})
}

func TestGetPayloadJSON(t *testing.T) {
	client := NewClient(time.Minute * 5)
	t.Run("populates passed struct with response payload from server endpoint", func(t *testing.T) {
		expected := TestEmployee{"employee", 5000}
		handler := getTestHandlerJSON("/", expected)
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
		handler := getTestHandlerJSON("/", payload)
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
