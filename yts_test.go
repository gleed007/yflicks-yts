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

func getTestServer(t *testing.T, pattern, name string) *httptest.Server {
	t.Helper()
	serveMux := &http.ServeMux{}
	serveMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		mockPath := path.Join("testdata", name)
		http.ServeFile(w, r, mockPath)
	})
	return httptest.NewServer(serveMux)
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

	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.DefaultTorrentTrackers() = %v, want %v", got, want)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	got := yts.DefaultClientConfig()
	want := yts.ClientConfig{
		APIBaseURL:      yts.DefaultAPIBaseURL,
		SiteURL:         yts.DefaultSiteURL,
		SiteDomain:      yts.DefaultSiteDomain,
		RequestTimeout:  time.Minute,
		TorrentTrackers: yts.DefaultTorrentTrackers(),
		Debug:           false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.DefaultClientConfig() = %v, want %v", got, want)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	type args struct {
		config *yts.ClientConfig
	}
	tests := []struct {
		name      string
		args      args
		wantErr   error
		wantPanic bool
	}{
		{
			name:      fmt.Sprintf(`panic() if config request timeout < %d`, yts.TimeoutLimitLower),
			args:      args{&yts.ClientConfig{RequestTimeout: time.Second}},
			wantErr:   yts.ErrInvalidClientConfig,
			wantPanic: true,
		},
		{
			name:      fmt.Sprintf(`panic() if config request timeout > %d`, yts.TimeoutLimitUpper),
			args:      args{&yts.ClientConfig{RequestTimeout: time.Hour}},
			wantErr:   yts.ErrInvalidClientConfig,
			wantPanic: true,
		},
		{
			name:      "no panic() if valid client config provided",
			args:      args{&yts.ClientConfig{RequestTimeout: time.Minute}},
			wantErr:   nil,
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
					t.Errorf("yts.NewClientWithConfig() unexpected panic with value %v", recovered)
					return
				}
				if err, _ := recovered.(error); !errors.Is(err, tt.wantErr) {
					t.Errorf("yts.NewClientWithConfig() unexpected panic with error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}()
			yts.NewClientWithConfig(tt.args.config)
		})
	}
}

func TestNewClient(t *testing.T) {
	got := yts.NewClient()
	defaultConfig := yts.DefaultClientConfig()
	want := yts.NewClientWithConfig(&defaultConfig)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.NewClient() = %v, want %v", got, want)
	}
}

func TestClient_SearchMovies(t *testing.T) {
	const queryTerm = "Oppenheimer (2023)"

	type fields struct {
		config yts.ClientConfig
	}
	type args struct {
		ctx     context.Context
		filters *yts.SearchMoviesFilters
	}
	type handler struct {
		pattern  string
		testdata string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		handler handler
		want    *yts.SearchMoviesResponse
		wantErr error
	}{
		{
			name:    "returns error for invalid search filters",
			fields:  fields{yts.DefaultClientConfig()},
			args:    args{context.Background(), &yts.SearchMoviesFilters{}},
			want:    nil,
			wantErr: yts.ErrFilterValidationFailure,
		},
		{
			name:    "returns search movies response for valid filters",
			fields:  fields{yts.DefaultClientConfig()},
			args:    args{context.Background(), yts.DefaultSearchMoviesFilter(queryTerm)},
			handler: handler{"/list_movies.json", "list_movies.json"},
			want: &yts.SearchMoviesResponse{
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler.pattern != "" {
				server := getTestServer(t, tt.handler.pattern, tt.handler.testdata)
				tt.fields.config.APIBaseURL = server.URL
				defer server.Close()
			}

			cfg := tt.fields.config
			c := yts.NewClientWithConfig(&cfg)
			got, err := c.SearchMovies(tt.args.ctx, tt.args.filters)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("yts.Client.SearchMovies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yts.Client.SearchMovies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetMovieDetails(t *testing.T) {
	const movieID = 57427

	type fields struct {
		config yts.ClientConfig
	}
	type args struct {
		ctx     context.Context
		movieID int
		filters *yts.MovieDetailsFilters
	}
	type handler struct {
		pattern  string
		testdata string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		handler handler
		want    *yts.MovieDetailsResponse
		wantErr error
	}{
		{
			name:    "returns error for invalid movieID",
			fields:  fields{yts.DefaultClientConfig()},
			args:    args{context.Background(), 0, &yts.MovieDetailsFilters{}},
			want:    nil,
			wantErr: yts.ErrFilterValidationFailure,
		},
		{
			name:    "returns movie details response for valid movieID",
			fields:  fields{yts.DefaultClientConfig()},
			args:    args{context.Background(), movieID, yts.DefaultMovieDetailsFilters()},
			handler: handler{"/movie_details.json", "movie_details.json"},
			want: &yts.MovieDetailsResponse{
				Data: yts.MovieDetailsData{
					Movie: yts.MovieDetails{
						MoviePartial: yts.MoviePartial{ID: movieID},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler.pattern != "" {
				server := getTestServer(t, tt.handler.pattern, tt.handler.testdata)
				tt.fields.config.APIBaseURL = server.URL
				defer server.Close()
			}

			cfg := tt.fields.config
			c := yts.NewClientWithConfig(&cfg)
			got, err := c.GetMovieDetails(tt.args.ctx, tt.args.movieID, tt.args.filters)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("yts.Client.GetMovieDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yts.Client.GetMovieDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetMovieSuggestions(t *testing.T) {
	const movieID = 57427

	type fields struct {
		config yts.ClientConfig
	}
	type args struct {
		ctx     context.Context
		movieID int
	}
	type handler struct {
		pattern  string
		testdata string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		handler handler
		want    *yts.MovieSuggestionsResponse
		wantErr error
	}{
		{
			name:    "returns error for invalid movieID",
			fields:  fields{yts.DefaultClientConfig()},
			args:    args{context.Background(), 0},
			want:    nil,
			wantErr: yts.ErrFilterValidationFailure,
		},
		{
			name:    "returns movie suggestions response for valid movieID",
			fields:  fields{yts.DefaultClientConfig()},
			args:    args{context.Background(), movieID},
			handler: handler{"/movie_suggestions.json", "movie_suggestions.json"},
			want: &yts.MovieSuggestionsResponse{
				Data: yts.MovieSuggestionsData{
					MovieCount: 0,
					Movies: []yts.Movie{
						{MoviePartial: yts.MoviePartial{ID: 2719}},
						{MoviePartial: yts.MoviePartial{ID: 53072}},
						{MoviePartial: yts.MoviePartial{ID: 55197}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler.pattern != "" {
				server := getTestServer(t, tt.handler.pattern, tt.handler.testdata)
				tt.fields.config.APIBaseURL = server.URL
				defer server.Close()
			}

			cfg := tt.fields.config
			c := yts.NewClientWithConfig(&cfg)
			got, err := c.GetMovieSuggestions(tt.args.ctx, tt.args.movieID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("yts.Client.GetMovieSuggestions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yts.Client.GetMovieSuggestions() = %v, want %v", got, tt.want)
			}
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.Client.GetMagnetLinks() = %v, want %v", got, want)
	}
}
