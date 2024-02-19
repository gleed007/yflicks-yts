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
		SiteDomain:      yts.DefaultSiteDomain,
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
		clientCfg *yts.ClientConfig
		wantErr   error
		wantPanic bool
	}{
		{
			name:      fmt.Sprintf(`panic() if config request timeout < %d`, yts.TimeoutLimitLower),
			clientCfg: &yts.ClientConfig{RequestTimeout: time.Second},
			wantErr:   yts.ErrInvalidClientConfig,
			wantPanic: true,
		},
		{
			name:      fmt.Sprintf(`panic() if config request timeout > %d`, yts.TimeoutLimitUpper),
			clientCfg: &yts.ClientConfig{RequestTimeout: time.Hour},
			wantErr:   yts.ErrInvalidClientConfig,
			wantPanic: true,
		},
		{
			name:      "no panic() if valid client config provided",
			clientCfg: &yts.ClientConfig{RequestTimeout: time.Minute},
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				if !tt.wantPanic && recovered == nil {
					return
				}
				if !tt.wantPanic && recovered != nil {
					t.Errorf("%s() unexpected panic with value %v", methodName, recovered)
					return
				}
				if err, _ := recovered.(error); !errors.Is(err, tt.wantErr) {
					t.Errorf("%s() unexpected panic with error = %v, wantErr %v", methodName, err, tt.wantErr)
					return
				}
			}()
			yts.NewClientWithConfig(tt.clientCfg)
		})
	}
}

func TestNewClient(t *testing.T) {
	defaultConfig := yts.DefaultClientConfig()
	got := yts.NewClient()
	want := yts.NewClientWithConfig(&defaultConfig)
	assertEqual(t, "NewClient", got, want)
}

type testHTTPHandlerConfig struct {
	filename string
	pattern  string
}

func getHandlerConfigFor(t *testing.T, pattern, dir, filename string) testHTTPHandlerConfig {
	t.Helper()
	return testHTTPHandlerConfig{
		filename: path.Join(dir, filename),
		pattern:  path.Join("/", pattern),
	}
}

func createTestServer(t *testing.T, config testHTTPHandlerConfig) *httptest.Server {
	t.Helper()
	serveMux := &http.ServeMux{}
	serveMux.HandleFunc(config.pattern, func(w http.ResponseWriter, r *http.Request) {
		mockPath := path.Join("testdata", config.filename)
		http.ServeFile(w, r, mockPath)
	})
	return httptest.NewServer(serveMux)
}

func TestClient_SearchMovies(t *testing.T) {
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
			name:       "returns mocked ok response for default filters",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.json"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			filters:    yts.DefaultSearchMoviesFilter(queryTerm),
			want:       mockedOKResponse,
		},
		{
			name:       "returns mocked ok response for valid filters",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.json"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			filters:    validSearchFilters,
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerCfg.filename != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.APIBaseURL = *serverURL
				defer server.Close()
			}

			c := yts.NewClientWithConfig(&clientCfg)
			got, err := c.SearchMovies(tt.ctx, tt.filters)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_GetMovieDetails(t *testing.T) {
	const (
		movieID     = 57427
		methodName  = "Client.GetMovieDetails"
		testdataDir = "get_movie_details"
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
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for negative movieID`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			movieID:   -1,
			wantErr:   yts.ErrFilterValidationFailure,
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
			name:       "returns mocked ok response for valid movieID",
			movieID:    movieID,
			clientCfg:  yts.DefaultClientConfig(),
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.json"),
			ctx:        context.Background(),
			filters:    yts.DefaultMovieDetailsFilters(),
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerCfg.filename != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.APIBaseURL = *serverURL
				defer server.Close()
			}

			c := yts.NewClientWithConfig(&clientCfg)
			got, err := c.GetMovieDetails(tt.ctx, tt.movieID, tt.filters)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_GetMovieSuggestions(t *testing.T) {
	const (
		movieID     = 57427
		methodName  = "Client.GetMovieSuggestions"
		testdataDir = "get_movie_suggestions"
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
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      `returns error for negative movieID`,
			clientCfg: yts.DefaultClientConfig(),
			ctx:       context.Background(),
			movieID:   -1,
			wantErr:   yts.ErrFilterValidationFailure,
		},
		{
			name:      "returns error when request context times out",
			clientCfg: yts.DefaultClientConfig(),
			ctx:       timedoutCtx,
			movieID:   movieID,
			wantErr:   context.DeadlineExceeded,
		},
		{
			name:       "returns mocked ok response for valid movieID",
			clientCfg:  yts.DefaultClientConfig(),
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.json"),
			ctx:        context.Background(),
			movieID:    movieID,
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		clientCfg := tt.clientCfg
		t.Run(tt.name, func(t *testing.T) {
			if tt.handlerCfg.filename != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.APIBaseURL = *serverURL
				defer server.Close()
			}

			c := yts.NewClientWithConfig(&clientCfg)
			got, err := c.GetMovieSuggestions(tt.ctx, tt.movieID)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_GetTrendingMovies(t *testing.T) {
	const (
		methodName  = "Client.GetTrendingMovies"
		testdataDir = "get_trending_movies"
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
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_selector.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Title" is missing from a scraped movie`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_title.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Year" is missing from scraped movie`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_year.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Link" is missing from scraped movie`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_link.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when "Image" is missing from scraped movie`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_image.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Rating" is invalid`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "invalid_rating.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Genres" are invalid`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "invalid_genres.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when request context times out",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        timedoutCtx,
			wantErr:    context.DeadlineExceeded,
		},
		{
			name:       "returns mocked ok response when scraping succeeds",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.html"),
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

			c := yts.NewClientWithConfig(&clientCfg)
			got, err := c.GetTrendingMovies(tt.ctx)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_GetHomePageContent(t *testing.T) {
	const (
		methodName  = "Client.GetHomePageContent"
		testdataDir = "get_homepage_content"
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
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_popular.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when latest torrents selector missing",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_latest.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when upcoming movies selector missing",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "missing_upcoming.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when validation for scraped popular movies fail",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "invalid_popular.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when validation for scraped latest torrents fail",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "invalid_latest.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when validation for scraped upcoming movies fail",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "invalid_upcoming.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Quality" is invalid`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "invalid_quality.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       `returns error when scraped "Progress" is invalid`,
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "invalid_progress.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			wantErr:    yts.ErrContentRetrievalFailure,
		},
		{
			name:       "returns error when request context times out",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        timedoutCtx,
			wantErr:    context.DeadlineExceeded,
		},
		{
			name:       "returns mocked ok response when scraping succeeds",
			handlerCfg: getHandlerConfigFor(t, pattern, testdataDir, "ok_response.html"),
			clientCfg:  yts.DefaultClientConfig(),
			ctx:        context.Background(),
			want:       mockedOKResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientCfg := tt.clientCfg
			if tt.handlerCfg.filename != "" {
				server := createTestServer(t, tt.handlerCfg)
				serverURL, _ := url.Parse(server.URL)
				clientCfg.SiteURL = *serverURL
				defer server.Close()
			}

			c := yts.NewClientWithConfig(&clientCfg)
			got, err := c.GetHomePageContent(tt.ctx)
			assertError(t, methodName, err, tt.wantErr)
			assertEqual(t, methodName, got, tt.want)
		})
	}
}

func TestClient_GetMagnetLinks(t *testing.T) {
	var (
		config   = yts.DefaultClientConfig()
		client   = yts.NewClientWithConfig(&config)
		trackers = url.Values{}
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
			strings.ToUpper(config.SiteDomain),
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

	got := client.GetMagnetLinks(&infoGetter)
	assertEqual(t, "Client.GetMagnetLinks", got, want)
}
