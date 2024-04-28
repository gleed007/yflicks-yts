package yts_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	yts "github.com/atifcppprogrammer/yflicks-yts"
)

func assertEqual(t *testing.T, method string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s() = %v, want %v", method, got, want)
	}
}

func assertError(t *testing.T, method string, gotErr, wantErr error) {
	t.Helper()
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("%s() error = %v, wantErr %v", method, gotErr, wantErr)
	}
}

func TestDefaultTorrentTrackers(t *testing.T) {
	got := yts.DefaultTorrentTrackers()
	want := []string{
		"udp://open.demonii.com:1337/announce",
		"udp://tracker.openbittorrent.com:80",
		"udp://tracker.coppersurfer.tk:6969",
		"udp://glotorrents.pw:6969/announce",
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://torrent.gresille.org:80/announce",
		"udp://p4p.arenabg.com:1337",
		"udp://tracker.leechers-paradise.org:6969",
	}

	assertEqual(t, "DefaultTorrentTrackers", got, want)
}

func TestDefaultClientConfig(t *testing.T) {
	var (
		parsedSiteURL, _    = url.Parse(yts.DefaultSiteURL)
		parsedAPIBaseURL, _ = url.Parse(yts.DefaultAPIBaseURL)
	)

	got := yts.DefaultClientConfig()
	want := yts.ClientConfig{
		APIBaseURL:      *parsedAPIBaseURL,
		SiteURL:         *parsedSiteURL,
		RequestTimeout:  time.Minute,
		TorrentTrackers: yts.DefaultTorrentTrackers(),
		Debug:           false,
	}

	assertEqual(t, "DefaultClientConfig", got, want)
}

func TestNewClientWithConfig(t *testing.T) {
	const methodName = "NewClientWithConfig"

	tests := []struct {
		name      string
		clientCfg yts.ClientConfig
		wantErr   error
	}{
		{
			name:      fmt.Sprintf(`returns error if config request timeout < %d`, yts.TimeoutLimitLower),
			clientCfg: yts.ClientConfig{RequestTimeout: time.Second},
			wantErr:   yts.ErrInvalidClientConfig,
		},
		{
			name:      fmt.Sprintf(`returns error if config request timeout > %d`, yts.TimeoutLimitUpper),
			clientCfg: yts.ClientConfig{RequestTimeout: time.Hour},
			wantErr:   yts.ErrInvalidClientConfig,
		},
		{
			name:      "returns nil error if valid client config provided",
			clientCfg: yts.ClientConfig{RequestTimeout: time.Minute},
			wantErr:   nil,
		},
		{
			name:      "returns nil error if default client config provided",
			clientCfg: yts.DefaultClientConfig(),
			wantErr:   nil,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			_, err := yts.NewClientWithConfig(&clientCfg)
			assertError(t, methodName, err, tt.wantErr)
		})
	}
}

func TestNewClient(t *testing.T) {
	var (
		defaultConfig = yts.DefaultClientConfig()
		got           = yts.NewClient()
		want, _       = yts.NewClientWithConfig(&defaultConfig)
	)
	assertEqual(t, "NewClient", got, want)
}

type testHTTPHandlerConfig struct {
	filename   string
	pattern    string
	statusCode int
}

func defaultHandlerConfig(t *testing.T, pattern, dir, filename string) testHTTPHandlerConfig {
	t.Helper()
	return testHTTPHandlerConfig{
		filename:   path.Join(dir, filename),
		pattern:    path.Join("/", pattern),
		statusCode: http.StatusOK,
	}
}

func handlerConfigWithStatusCode(t *testing.T, pattern string, statusCode int) testHTTPHandlerConfig {
	t.Helper()
	return testHTTPHandlerConfig{
		pattern:    path.Join("/", pattern),
		statusCode: statusCode,
	}
}

func createTestServer(t *testing.T, config testHTTPHandlerConfig) *httptest.Server {
	t.Helper()
	serveMux := &http.ServeMux{}
	serveMux.HandleFunc(config.pattern, func(w http.ResponseWriter, r *http.Request) {
		switch config.statusCode {
		case http.StatusOK:
			mockPath := path.Join("testdata", config.filename)
			http.ServeFile(w, r, mockPath)
		default:
			w.WriteHeader(config.statusCode)
			fmt.Fprintf(w, "status_code: %d", config.statusCode)
		}
	})
	return httptest.NewServer(serveMux)
}

func TestClient_SearchMoviesWithContext(t *testing.T) {
	const (
		queryTerm   = "Oppenheimer (2023)"
		methodName  = "Client.SearchMovies"
		testdataDir = "search_movies"
		pattern     = "list_movies.json"
	)

	const (
		vLt = 10
		vPg = 1
		vQl = yts.Quality1080p
		vMr = 9
		vQt = queryTerm
		vGr = yts.GenreAnimation
		vSb = yts.SortByDownloadCount
		vOb = yts.OrderByAsc
		vWr = false
	)

	timedoutCtx, cancel := context.WithDeadline(
		context.Background(), time.Now(),
	)
	defer cancel()

	validSearchFilters := &yts.SearchMoviesFilters{
		Limit:         vLt,
		Page:          vPg,
		Quality:       vQl,
		MinimumRating: vMr,
		QueryTerm:     vQt,
		Genre:         vGr,
		SortBy:        vSb,
		OrderBy:       vOb,
		WithRTRatings: vWr,
	}

	mockedOKResponse := &yts.SearchMoviesResponse{
		Data: yts.SearchMoviesData{
			MovieCount: 3,
			PageNumber: 1,
			Limit:      20,
			Movies: []yts.Movie{
				{MoviePartial: yts.MoviePartial{ID: 57427}},
				{MoviePartial: yts.MoviePartial{ID: 57795}},
				{MoviePartial: yts.MoviePartial{ID: 53181}},
			},
		},
	}

	tests := []struct {
		name       string
		handlerCfg testHTTPHandlerConfig
		clientCfg  yts.ClientConfig
		ctx        context.Context
		filters    *yts.SearchMoviesFilters
		want       *yts.SearchMoviesResponse
		wantErr    error
	}{
		{
			name:      `returns error for "0" value search filters`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid minimum "Limit" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{-1, vPg, vQl, vMr, vQt, vGr, vSb, vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid maximum "Limit" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{51, vPg, vQl, vMr, vQt, vGr, vSb, vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid minimum "Page" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{vLt, -1, vQl, vMr, vQt, vGr, vSb, vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid "Quality" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{vLt, vPg, "invalid", vMr, vQt, vGr, vSb, vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid minimum "MinimumRating" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{vLt, vPg, vQl, -1, vQt, vGr, vSb, vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid maximum "MinimumRating" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{vLt, vPg, vQl, 10, vQt, vGr, vSb, vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid "Genre" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{vLt, vPg, vQl, vMr, vQt, "invalid", vSb, vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid "SortBy" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{vLt, vPg, vQl, vMr, vQt, vGr, "invalid", vOb, vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for invalid "OrderBy" filter`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.SearchMoviesFilters{vLt, vPg, vQl, vMr, vQt, vGr, vSb, "invalid", vWr},
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      "returns error when request context times out",
			clientCfg: yts.DefaultClientConfig(),
			ctx:       timedoutCtx,
			filters:   validSearchFilters,
			wantErr:   context.DeadlineExceeded,
		},
		{
			name:       "returns error when response status is outside 2.x.x range",
			handlerCfg: handlerConfigWithStatusCode(t, pattern, http.StatusNotFound),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			filters:    validSearchFilters,
			wantErr:    yts.ErrUnexpectedHTTPResponseStatus,
		},
		{
			name:       "returns mocked ok response for default filters",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.json"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			filters:    yts.DefaultSearchMoviesFilters(queryTerm),
			want:       mockedOKResponse,
		},
		{
			name:       "returns mocked ok response for valid filters",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.json"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			filters:    validSearchFilters,
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerCfg.pattern != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.APIBaseURL = *serverURL
				defer server.Close()
			}

			c, _ := yts.NewClientWithConfig(&clientCfg)
			got, err := c.SearchMoviesWithContext(tt.ctx, tt.filters)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_MovieDetailsWithContext(t *testing.T) {
	const (
		movieID     = 57427
		methodName  = "Client.MovieDetails"
		testdataDir = "movie_details"
		pattern     = "movie_details.json"
	)

	timedoutCtx, cancel := context.WithDeadline(
		context.Background(), time.Now(),
	)
	defer cancel()

	mockedOKResponse := &yts.MovieDetailsResponse{
		Data: yts.MovieDetailsData{
			Movie: yts.MovieDetails{
				MoviePartial: yts.MoviePartial{ID: movieID},
			},
		},
	}

	tests := []struct {
		name       string
		handlerCfg testHTTPHandlerConfig
		clientCfg  yts.ClientConfig
		ctx        context.Context
		movieID    int
		filters    *yts.MovieDetailsFilters
		want       *yts.MovieDetailsResponse
		wantErr    error
	}{
		{
			name:      `returns error for "0" movieID`,
			movieID:   0,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			filters:   &yts.MovieDetailsFilters{},
			wantErr:   yts.ErrValidationFailure,
		},
		{
			name:      `returns error for negative movieID`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			movieID:   -1,
			wantErr:   yts.ErrValidationFailure,
		},
		{
			name:      "returns error when request context times out",
			clientCfg: yts.DefaultClientConfig(),
			ctx:       timedoutCtx,
			movieID:   movieID,
			filters:   yts.DefaultMovieDetailsFilters(),
			wantErr:   context.DeadlineExceeded,
		},
		{
			name:       "returns error when response status is outside 2.x.x range",
			handlerCfg: handlerConfigWithStatusCode(t, pattern, http.StatusNotFound),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieID:    movieID,
			filters:    yts.DefaultMovieDetailsFilters(),
			wantErr:    yts.ErrUnexpectedHTTPResponseStatus,
		},
		{
			name:       "returns mocked ok response for valid movieID",
			movieID:    movieID,
			clientCfg:  yts.DefaultClientConfig(),
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.json"),
			ctx:        context.Background(),
			filters:    yts.DefaultMovieDetailsFilters(),
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerCfg.pattern != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.APIBaseURL = *serverURL
				defer server.Close()
			}

			c, _ := yts.NewClientWithConfig(&clientCfg)
			got, err := c.MovieDetailsWithContext(tt.ctx, tt.movieID, tt.filters)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_MovieSuggestionsWithContext(t *testing.T) {
	const (
		movieID     = 57427
		methodName  = "Client.MovieSuggestions"
		testdataDir = "movie_suggestions"
		pattern     = "movie_suggestions.json"
	)

	timedoutCtx, cancel := context.WithDeadline(
		context.Background(), time.Now(),
	)
	defer cancel()

	mockedOKResponse := &yts.MovieSuggestionsResponse{
		Data: yts.MovieSuggestionsData{
			MovieCount: 0,
			Movies: []yts.Movie{
				{MoviePartial: yts.MoviePartial{ID: 2719}},
				{MoviePartial: yts.MoviePartial{ID: 53072}},
				{MoviePartial: yts.MoviePartial{ID: 55197}},
			},
		},
	}

	tests := []struct {
		name       string
		handlerCfg testHTTPHandlerConfig
		clientCfg  yts.ClientConfig
		ctx        context.Context
		movieID    int
		want       *yts.MovieSuggestionsResponse
		wantErr    error
	}{
		{
			name:      `returns error for "0" movieID`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			movieID:   0,
			wantErr:   yts.ErrValidationFailure,
		},
		{
			name:      `returns error for negative movieID`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			movieID:   -1,
			wantErr:   yts.ErrValidationFailure,
		},
		{
			name:      "returns error when request context times out",
			clientCfg: yts.DefaultClientConfig(),
			ctx:       timedoutCtx,
			movieID:   movieID,
			wantErr:   context.DeadlineExceeded,
		},
		{
			name:       "returns error when response status is outside 2.x.x range",
			handlerCfg: handlerConfigWithStatusCode(t, pattern, http.StatusNotFound),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieID:    movieID,
			wantErr:    yts.ErrUnexpectedHTTPResponseStatus,
		},
		{
			name:       "returns mocked ok response for valid movieID",
			clientCfg:  yts.DefaultClientConfig(),
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.json"),
			ctx:        context.Background(),
			movieID:    movieID,
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerCfg.pattern != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.APIBaseURL = *serverURL
				defer server.Close()
			}

			c, _ := yts.NewClientWithConfig(&clientCfg)
			got, err := c.MovieSuggestionsWithContext(tt.ctx, tt.movieID)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_TrendingMoviesWithContext(t *testing.T) {
	const (
		methodName  = "Client.TrendingMovies"
		testdataDir = "trending_movies"
		pattern     = "/"
	)

	timedoutCtx, cancel := context.WithDeadline(
		context.Background(), time.Now(),
	)
	defer cancel()

	mockedOKResponse := &yts.TrendingMoviesResponse{
		Data: yts.TrendingMoviesData{
			Movies: []yts.SiteMovie{{
				Rating: "7.6 / 10",
				SiteMovieBase: yts.SiteMovieBase{
					Title:  "Superbad",
					Year:   2007,
					Link:   "https://yts.mx/movies/superbad-2007",
					Image:  "/assets/images/movies/Superbad_2007/medium-cover.jpg",
					Genres: []yts.Genre{"Action", "Comedy"},
				},
			}},
		},
	}

	tests := []struct {
		name       string
		handlerCfg testHTTPHandlerConfig
		clientCfg  yts.ClientConfig
		ctx        context.Context
		want       *yts.TrendingMoviesResponse
		wantErr    error
	}{
		{
			name:       "returns error when trending movies selector missing",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_selector.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Title" is missing from a scraped movie`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_title.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Year" is missing from scraped movie`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_year.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Link" is missing from scraped movie`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_link.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Image" is missing from scraped movie`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_image.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Year" is invalid`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_year.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Rating" is invalid`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_rating.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Genres" are invalid`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_genres.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when request context times out",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        timedoutCtx,
			wantErr:    context.DeadlineExceeded,
		},
		{
			name:       "returns error when response status is outside 2.x.x range",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "non_existent.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrUnexpectedHTTPResponseStatus,
		},
		{
			name:       "returns mocked ok response when scraping succeeds",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerCfg.pattern != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.SiteURL = *serverURL
				defer server.Close()
			}

			c, _ := yts.NewClientWithConfig(&clientCfg)
			got, err := c.TrendingMoviesWithContext(tt.ctx)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_HomePageContentWithContext(t *testing.T) {
	const (
		methodName  = "Client.HomePageContent"
		testdataDir = "homepage_content"
		pattern     = "/"
	)

	timedoutCtx, cancel := context.WithDeadline(
		context.Background(), time.Now(),
	)
	defer cancel()

	mockedOKResponse := &yts.HomePageContentResponse{
		Data: yts.HomePageContentData{
			Popular: []yts.SiteMovie{{
				Rating: "6.8 / 10",
				SiteMovieBase: yts.SiteMovieBase{
					Title:  "Migration",
					Year:   2023,
					Link:   "https://yts.mx/movies/migration-2023",
					Image:  "/assets/images/movies/migration_2023/medium-cover.jpg",
					Genres: []yts.Genre{"Action", "Adventure"},
				},
			}},
			Latest: []yts.SiteMovie{{
				Rating: "5.3 / 10",
				SiteMovieBase: yts.SiteMovieBase{
					Title:  "[NL] Het einde van de reis",
					Year:   1981,
					Link:   "https://yts.mx/movies/het-einde-van-de-reis-1981",
					Image:  "/assets/images/movies/het_einde_van_de_reis_1981/medium-cover.jpg",
					Genres: []yts.Genre{"Action"},
				},
			}},
			Upcoming: []yts.SiteUpcomingMovie{{
				Progress: 28,
				Quality:  yts.Quality2160p,
				SiteMovieBase: yts.SiteMovieBase{
					Title:  "Boyz n the Hood",
					Year:   1991,
					Link:   "https://www.imdb.com/title/tt0101507/",
					Image:  "/assets/images/movies/Boyz_n_the_Hood_1991/medium-cover.jpg",
					Genres: []yts.Genre{},
				},
			}},
		},
	}

	tests := []struct {
		name       string
		handlerCfg testHTTPHandlerConfig
		clientCfg  yts.ClientConfig
		ctx        context.Context
		want       *yts.HomePageContentResponse
		wantErr    error
	}{
		{
			name:       "returns error when popular movies selector missing",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_popular.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when latest torrents selector missing",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_latest.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when upcoming movies selector missing",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_upcoming.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when validation for scraped popular movies fail",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_popular.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when validation for scraped latest torrents fail",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_latest.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when validation for scraped upcoming movies fail",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_upcoming.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Quality" is invalid`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_quality.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Progress" is invalid`,
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_progress.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when request context times out",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        timedoutCtx,
			wantErr:    context.DeadlineExceeded,
		},
		{
			name:       "returns error when response status is outside 2.x.x range",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "non_existent.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrUnexpectedHTTPResponseStatus,
		},
		{
			name:       "returns mocked ok response when scraping succeeds",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientCfg := tt.clientCfg
			if tt.handlerCfg.pattern != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.SiteURL = *serverURL
				defer server.Close()
			}

			c, _ := yts.NewClientWithConfig(&clientCfg)
			got, err := c.HomePageContentWithContext(tt.ctx)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_ResolveMovieSlugToIDWithContext(t *testing.T) {
	const (
		methodName  = "Client.ResolveMovieSlugtoID"
		testdataDir = "resolve_movie_slug"
		movieID     = 3175
		movieSlug   = "the-dark-knight-2008"
		pattern     = "/movies/the-dark-knight-2008"
	)

	tests := []struct {
		name       string
		handlerCfg testHTTPHandlerConfig
		clientCfg  yts.ClientConfig
		ctx        context.Context
		movieSlug  string
		want       int
		wantErr    error
	}{
		{
			name:       "resolves movie slug to ID successfully when available",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieSlug:  "the-dark-knight-2008",
			want:       movieID,
		},
		{
			name:       "returns error when required selector is not available",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_selector.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieSlug:  "the-dark-knight-2008",
			want:       0,
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error if fails to pass available movie ID",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_id.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieSlug:  "the-dark-knight-2008",
			want:       0,
			wantErr:    yts.ErrContentRetrievalFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientCfg := tt.clientCfg
			if tt.handlerCfg.pattern != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.SiteURL = *serverURL
				defer server.Close()
			}

			c, _ := yts.NewClientWithConfig(&clientCfg)
			got, err := c.ResolveMovieSlugToIDWithContext(tt.ctx, tt.movieSlug)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_MovieDirectorWithContext(t *testing.T) {
	const (
		methodName  = "Client.MovieDirector"
		testdataDir = "get_movie_director"
		movieSlug   = "road-house-1989"
		pattern     = "/movies/road-house-1989"
	)

	timedoutCtx, cancel := context.WithDeadline(
		context.Background(), time.Now(),
	)
	defer cancel()

	mockedOKResponse := yts.MovieDirectorResponse{
		Data: yts.MovieDirectorData{
			Director: yts.SiteMovieDirector{
				Name:          "Rowdy Herrington",
				URLSmallImage: "https://img.yts.mx/assets/images/actors/thumb/nm1509613.jpg",
			},
		},
	}

	tests := []struct {
		name       string
		handlerCfg testHTTPHandlerConfig
		clientCfg  yts.ClientConfig
		ctx        context.Context
		movieSlug  string
		want       *yts.MovieDirectorResponse
		wantErr    error
	}{
		{
			name:      "returns error when movie slug is an empty string",
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			movieSlug: "",
			wantErr:   yts.ErrValidationFailure,
		},
		{
			name:       "returns error when director selector is missing",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_director.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieSlug:  movieSlug,
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when director name is missing",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "missing_name.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieSlug:  movieSlug,
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when director thumbnail URL is missing",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "invalid_thumbnail.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieSlug:  movieSlug,
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when request context times out",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        timedoutCtx,
			movieSlug:  movieSlug,
			wantErr:    context.DeadlineExceeded,
		},
		{
			name:       "returns error when response status is outside 2.x.x range",
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "non_existant.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			movieSlug:  movieSlug,
			wantErr:    yts.ErrUnexpectedHTTPResponseStatus,
		},
		{
			name:       "returns mocked ok response when scraping succeeds",
			clientCfg:  yts.DefaultClientConfig(),
			handlerCfg: defaultHandlerConfig(t, pattern, testdataDir, "ok_response.html"),
			ctx:        context.Background(),
			movieSlug:  movieSlug,
			want:       &mockedOKResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientCfg := tt.clientCfg
			if tt.handlerCfg.pattern != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.SiteURL = *serverURL
				defer server.Close()
			}

			c, _ := yts.NewClientWithConfig(&clientCfg)
			got, err := c.MovieDirectorWithContext(tt.ctx, tt.movieSlug)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_MagnetLinks(t *testing.T) {
	var (
		config    = yts.DefaultClientConfig()
		client, _ = yts.NewClientWithConfig(&config)
		trackers  = url.Values{}
	)

	for _, tracker := range config.TorrentTrackers {
		trackers.Add("tr", tracker)
	}

	infoGetter := yts.MoviePartial{
		TitleLong: "Oppenheimer (2023)",
		Torrents: []yts.Torrent{
			{Hash: "Hash0", Quality: yts.Quality720p},
			{Hash: "Hash1", Quality: yts.Quality1080p},
			{Hash: "Hash2", Quality: yts.Quality1080p},
			{Hash: "Hash3", Quality: yts.Quality2160p},
		},
	}

	getMagnetFor := func(torrent yts.Torrent) string {
		torrentName := fmt.Sprintf(
			"%s+[%s]+[%s]",
			infoGetter.GetTorrentInfo().MovieTitle,
			torrent.Quality,
			strings.ToUpper(config.SiteURL.Host),
		)

		return fmt.Sprintf(
			"magnet:?xt=urn:btih:%s&dn=%s&%s",
			torrent.Hash,
			url.QueryEscape(torrentName),
			trackers.Encode(),
		)
	}

	want := make(yts.TorrentMagnets, 0)
	torrents := infoGetter.GetTorrentInfo().Torrents
	for i := 0; i < len(torrents); i++ {
		want[torrents[i].Quality] = getMagnetFor(torrents[i])
	}

	got := client.MagnetLinks(&infoGetter)
	assertEqual(t, "Client.MagnetLinks", got, want)
}
