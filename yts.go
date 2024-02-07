package yts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultAPIBaseURL = "https://yts.mx/api/v2"
	DefaultSiteURL    = "https://yts.mx"
	DefaultSiteDomain = "yts.mx"
)

const (
	TimeoutLimitUpper = 5 * time.Minute
	TimeoutLimitLower = 5 * time.Second
)

var debug = newLogger()

type ClientConfig struct {
	APIBaseURL      string
	SiteURL         string
	SiteDomain      string
	TorrentTrackers []string
	RequestTimeout  time.Duration
	Debug           bool
}

type Client struct {
	config    ClientConfig
	netClient *http.Client
}

var (
	ErrInvalidClientConfig     = errors.New("invalid_client_config")
	ErrContentRetrievalFailure = errors.New("content_retrieval_failure")
	ErrFilterValidationFailure = errors.New("filter_validation_failure")
)

func wrapErr(sentinel error, others ...error) error {
	return fmt.Errorf("%w: %s", sentinel, errors.Join(others...))
}

func DefaultTorrentTrackers() []string {
	return []string{
		"udp://open.demonii.com:1337/announce",
		"udp://tracker.openbittorrent.com:80",
		"udp://tracker.coppersurfer.tk:6969",
		"udp://glotorrents.pw:6969/announce",
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://torrent.gresille.org:80/announce",
		"udp://p4p.arenabg.com:1337",
		"udp://tracker.leechers-paradise.org:6969",
	}
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		APIBaseURL:      DefaultAPIBaseURL,
		SiteURL:         DefaultSiteURL,
		SiteDomain:      DefaultSiteDomain,
		RequestTimeout:  time.Minute,
		TorrentTrackers: DefaultTorrentTrackers(),
		Debug:           false,
	}
}

func NewClientWithConfig(config *ClientConfig) *Client {
	if config.RequestTimeout < TimeoutLimitLower {
		err := fmt.Errorf(
			"request timeout must be >= %s, you provided %q",
			TimeoutLimitLower,
			config.RequestTimeout,
		)
		panic(wrapErr(ErrInvalidClientConfig, err))
	}

	if TimeoutLimitUpper < config.RequestTimeout {
		err := fmt.Errorf(
			"request timeout must be <= %s, you provided %q",
			TimeoutLimitUpper,
			config.RequestTimeout,
		)
		panic(wrapErr(ErrInvalidClientConfig, err))
	}

	if config.Debug {
		debug.setDebug(true)
	}

	netClient := &http.Client{Timeout: config.RequestTimeout}
	return &Client{*config, netClient}
}

func NewClient() *Client {
	defaultConfig := DefaultClientConfig()
	return NewClientWithConfig(&defaultConfig)
}

type SearchMoviesData struct {
	MovieCount int     `json:"movie_count"`
	Limit      int     `json:"limit"`
	PageNumber int     `json:"page_number"`
	Movies     []Movie `json:"movies"`
}

type BaseResponse struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Meta          `json:"@meta"`
}

type SearchMoviesResponse struct {
	BaseResponse
	Data SearchMoviesData `json:"data"`
}

func (c *Client) SearchMovies(ctx context.Context, filters *SearchMoviesFilters) (
	*SearchMoviesResponse, error,
) {
	queryString, err := filters.getQueryString()
	if err != nil {
		return nil, wrapErr(ErrFilterValidationFailure, err)
	}

	parsedPayload := &SearchMoviesResponse{}
	targetURL := c.getAPIEndpoint("list_movies.json", queryString)
	err = c.newJSONRequestWithContext(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

type MovieDetailsData struct {
	Movie MovieDetails `json:"movie"`
}

type MovieDetailsResponse struct {
	BaseResponse
	Data MovieDetailsData `json:"data"`
}

func (c *Client) GetMovieDetails(ctx context.Context, movieID int, filters *MovieDetailsFilters) (
	*MovieDetailsResponse, error,
) {
	if movieID <= 0 {
		err := fmt.Errorf("provided movieID must be at least 1")
		return nil, wrapErr(ErrFilterValidationFailure, err)
	}

	queryString := fmt.Sprintf("movie_id=%d", movieID)
	if q := filters.getQueryString(); q != "" {
		queryString = fmt.Sprintf("movie_id=%d&%s", movieID, q)
	}

	parsedPayload := &MovieDetailsResponse{}
	targetURL := c.getAPIEndpoint("movie_details.json", queryString)
	err := c.newJSONRequestWithContext(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

type MovieSuggestionsData struct {
	MovieCount int     `json:"movie_count"`
	Movies     []Movie `json:"movies"`
}

type MovieSuggestionsResponse struct {
	BaseResponse
	Data MovieSuggestionsData `json:"data"`
}

func (c *Client) GetMovieSuggestions(ctx context.Context, movieID int) (
	*MovieSuggestionsResponse, error,
) {
	if movieID <= 0 {
		err := fmt.Errorf("provided movieID must be at least 1")
		return nil, wrapErr(ErrFilterValidationFailure, err)
	}

	var (
		movieIDStr  = fmt.Sprintf("%d", movieID)
		queryValues = url.Values{"movie_id": []string{movieIDStr}}
		queryString = queryValues.Encode()
	)

	parsedPayload := &MovieSuggestionsResponse{}
	targetURL := c.getAPIEndpoint("movie_suggestions.json", queryString)
	err := c.newJSONRequestWithContext(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

type TrendingMoviesData struct {
	Movies []SiteMovie `json:"movies"`
}

type TrendingMoviesResponse struct {
	Data TrendingMoviesData `json:"data"`
}

func (c *Client) GetTrendingMovies(ctx context.Context) (
	*TrendingMoviesResponse, error,
) {
	pageURL := fmt.Sprintf("%s/trending-movies", c.config.SiteURL)
	response, err := c.newRequestWithContext(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	data, err := c.scrapeTrendingMoviesData(response.Body)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	return &TrendingMoviesResponse{*data}, nil
}

type HomePageContentData struct {
	Popular  []SiteMovie
	Latest   []SiteMovie
	Upcoming []SiteUpcomingMovie
}

type HomePageContentResponse struct {
	Data HomePageContentData `json:"data"`
}

func (c *Client) GetHomePageContent(ctx context.Context) (
	*HomePageContentResponse, error,
) {
	response, err := c.newRequestWithContext(ctx, c.config.SiteURL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	data, err := c.scrapeHomePageContentData(response.Body)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	return &HomePageContentResponse{*data}, nil
}

type TorrentMagnets map[Quality]string

func (c *Client) GetMagnetLinks(t TorrentInfoGetter) TorrentMagnets {
	var trackers = url.Values{}
	for _, tracker := range c.config.TorrentTrackers {
		trackers.Add("tr", tracker)
	}

	getMagnetFor := func(torrent Torrent) string {
		torrentName := fmt.Sprintf(
			"%s+[%s]+[%s]",
			t.GetTorrentInfo().MovieTitle,
			torrent.Quality,
			strings.ToUpper(c.config.SiteDomain),
		)

		return fmt.Sprintf(
			"magnet:?xt=urn:btih:%s&dn=%s&%s",
			torrent.Hash,
			url.QueryEscape(torrentName),
			trackers.Encode(),
		)
	}

	magnets := make(TorrentMagnets, 0)
	torrents := t.GetTorrentInfo().Torrents
	for i := 0; i < len(torrents); i++ {
		magnets[torrents[i].Quality] = getMagnetFor(torrents[i])
	}

	return magnets
}
