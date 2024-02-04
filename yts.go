package yts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type Client struct {
	APIBaseURL      string
	SiteURL         string
	SiteDomain      string
	NetClient       *http.Client
	TorrentTrackers []string
}

type BaseResponse struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Meta          `json:"@meta"`
}

var (
	ErrSiteScrapingFailure    = errors.New("invalid_site_scraping_failure")
	ErrQualityTorrentNotFound = errors.New("quality_torrent_not_found")
)

var ErrInvalidClientTimeout = fmt.Errorf(
	`invalid_client_timeout: "yts client timeout must be between %s and %s inclusive"`,
	TimeoutLimitLower,
	TimeoutLimitUpper,
)

func wrapErr(sentinel error, others ...error) error {
	return fmt.Errorf("%w: %s", sentinel, errors.Join(others...))
}

func NewClient(timeout time.Duration) *Client {
	if timeout < TimeoutLimitLower || TimeoutLimitUpper < timeout {
		panic(ErrInvalidClientTimeout)
	}

	return &Client{
		DefaultAPIBaseURL,
		DefaultSiteURL,
		DefaultSiteDomain,
		&http.Client{Timeout: timeout},
		DefaultTorrentTrackers(),
	}
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

type SearchMoviesData struct {
	MovieCount int     `json:"movie_count"`
	Limit      int     `json:"limit"`
	PageNumber int     `json:"page_number"`
	Movies     []Movie `json:"movies"`
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
		return nil, err
	}

	parsedPayload := &SearchMoviesResponse{}
	targetURL := c.getEndpointURL("list_movies.json", queryString)
	err = c.getPayloadJSON(ctx, targetURL, parsedPayload)
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
		return nil, wrapErr(ErrValidationFailure, err)
	}

	queryString := fmt.Sprintf("movie_id=%d", movieID)
	if q := filters.getQueryString(); q != "" {
		queryString = fmt.Sprintf("movie_id=%d&%s", movieID, q)
	}

	parsedPayload := &MovieDetailsResponse{}
	targetURL := c.getEndpointURL("movie_details.json", queryString)
	err := c.getPayloadJSON(ctx, targetURL, parsedPayload)
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
		return nil, wrapErr(ErrValidationFailure, err)
	}

	var (
		movieIDStr  = fmt.Sprintf("%d", movieID)
		queryValues = url.Values{"movie_id": []string{movieIDStr}}
		queryString = queryValues.Encode()
	)

	parsedPayload := &MovieSuggestionsResponse{}
	targetURL := c.getEndpointURL("movie_suggestions.json", queryString)
	err := c.getPayloadJSON(ctx, targetURL, parsedPayload)
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
	var rawPayload []byte
	pageURL := fmt.Sprintf("%s/trending-movies", c.SiteURL)
	rawPayload, err := c.getPayloadRaw(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	reader := strings.NewReader(string(rawPayload))
	data, err := c.scrapeTrendingMoviesData(reader)
	if err != nil {
		return nil, err
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
	var rawPayload []byte
	rawPayload, err := c.getPayloadRaw(ctx, c.SiteURL)
	if err != nil {
		return nil, err
	}

	reader := strings.NewReader(string(rawPayload))
	data, err := c.scrapeHomePageContentData(reader)
	if err != nil {
		return nil, err
	}

	return &HomePageContentResponse{*data}, nil
}

func (c *Client) GetMagnetLink(t TorrentInfoGetter, q Quality) (string, error) {
	var (
		foundTorrent = Torrent{}
		torrentInfo  = t.GetTorrentInfo()
	)

	for index := 0; index < len(torrentInfo.Torrents); index++ {
		if torrentInfo.Torrents[index].Quality == q {
			foundTorrent = torrentInfo.Torrents[index]
		}
	}

	if foundTorrent.Quality == "" {
		err := fmt.Errorf("no torrent found having quality %s", q)
		return "", wrapErr(ErrQualityTorrentNotFound, err)
	}

	torrentName := fmt.Sprintf(
		"%s+[%s]+[%s]",
		torrentInfo.MovieTitle, q, strings.ToUpper(c.SiteDomain),
	)

	var trackers = url.Values{}
	for _, tracker := range c.TorrentTrackers {
		trackers.Add("tr", tracker)
	}

	magnet := fmt.Sprintf(
		"magnet:?xt=urn:btih:%s&dn=%s&%s",
		foundTorrent.Hash, url.QueryEscape(torrentName), trackers.Encode(),
	)

	return magnet, nil
}

func (c *Client) getPayloadJSON(
	ctx context.Context, targetURL string, payload interface{},
) error {
	rawPayload, err := c.getPayloadRaw(ctx, targetURL)
	if err != nil {
		return err
	}

	err = json.Unmarshal(rawPayload, payload)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getPayloadRaw(ctx context.Context, targetURL string) (
	[]byte, error,
) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	parsed := parsedURL.String()
	request, err := http.NewRequestWithContext(ctx, "GET", parsed, http.NoBody)
	if err != nil {
		return nil, err
	}

	response, err := c.NetClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	rawPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return rawPayload, nil
}

func (c *Client) getEndpointURL(path, query string) string {
	targetURL := fmt.Sprintf("%s/%s", c.APIBaseURL, path)
	if query == "" {
		return targetURL
	}

	return fmt.Sprintf("%s?%s", targetURL, query)
}
