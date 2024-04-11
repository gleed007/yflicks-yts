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
)

const (
	TimeoutLimitUpper = 5 * time.Minute
	TimeoutLimitLower = 5 * time.Second
)

var debug = newLogger()

type ClientConfig struct {
	APIBaseURL      url.URL
	SiteURL         url.URL
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
	ErrValidationFailure       = errors.New("validation_failure")
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
	var (
		parsedSiteURL, _    = url.Parse(DefaultSiteURL)
		parsedAPIBaseURL, _ = url.Parse(DefaultAPIBaseURL)
	)

	return ClientConfig{
		APIBaseURL:      *parsedAPIBaseURL,
		SiteURL:         *parsedSiteURL,
		RequestTimeout:  time.Minute,
		TorrentTrackers: DefaultTorrentTrackers(),
		Debug:           false,
	}
}

func NewClientWithConfig(config *ClientConfig) (*Client, error) {
	if config.RequestTimeout < TimeoutLimitLower {
		err := fmt.Errorf(
			"request timeout must be >= %s, you provided %q",
			TimeoutLimitLower,
			config.RequestTimeout,
		)
		return nil, wrapErr(ErrInvalidClientConfig, err)
	}

	if TimeoutLimitUpper < config.RequestTimeout {
		err := fmt.Errorf(
			"request timeout must be <= %s, you provided %q",
			TimeoutLimitUpper,
			config.RequestTimeout,
		)
		return nil, wrapErr(ErrInvalidClientConfig, err)
	}

	if config.Debug {
		debug.setDebug(true)
	}

	netClient := &http.Client{Timeout: config.RequestTimeout}
	return &Client{*config, netClient}, nil
}

func NewClient() *Client {
	var (
		defaultConfig = DefaultClientConfig()
		client, _     = NewClientWithConfig(&defaultConfig)
	)
	return client
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

func (c *Client) SearchMoviesWithContext(ctx context.Context, filters *SearchMoviesFilters) (
	*SearchMoviesResponse, error,
) {
	queryString, err := filters.getQueryString()
	if err != nil {
		return nil, wrapErr(ErrFilterValidationFailure, err)
	}

	parsedPayload := &SearchMoviesResponse{}
	targetURLString := c.getAPIEndpoint("list_movies.json", queryString)
	targetURL, _ := url.Parse(targetURLString)
	err = c.newJSONRequestWithContext(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c *Client) SearchMovies(filters *SearchMoviesFilters) (*SearchMoviesResponse, error) {
	return c.SearchMoviesWithContext(context.Background(), filters)
}

type MovieDetailsData struct {
	Movie MovieDetails `json:"movie"`
}

type MovieDetailsResponse struct {
	BaseResponse
	Data MovieDetailsData `json:"data"`
}

func (c *Client) MovieDetailsWithContext(ctx context.Context, movieID int, filters *MovieDetailsFilters) (
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
	targetURLString := c.getAPIEndpoint("movie_details.json", queryString)
	targetURL, _ := url.Parse(targetURLString)
	err := c.newJSONRequestWithContext(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c *Client) MovieDetails(movieID int, filters *MovieDetailsFilters) (*MovieDetailsResponse, error) {
	return c.MovieDetailsWithContext(context.Background(), movieID, filters)
}

type MovieSuggestionsData struct {
	MovieCount int     `json:"movie_count"`
	Movies     []Movie `json:"movies"`
}

type MovieSuggestionsResponse struct {
	BaseResponse
	Data MovieSuggestionsData `json:"data"`
}

func (c *Client) MovieSuggestionsWithContext(ctx context.Context, movieID int) (
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
	targetURLString := c.getAPIEndpoint("movie_suggestions.json", queryString)
	targetURL, _ := url.Parse(targetURLString)
	err := c.newJSONRequestWithContext(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c *Client) MovieSuggestions(movieID int) (*MovieSuggestionsResponse, error) {
	return c.MovieSuggestionsWithContext(context.Background(), movieID)
}

func (c *Client) ResolveMovieSlugToIDWithContext(ctx context.Context, movieSlug string) (int, error) {
	if movieSlug == "" {
		err := fmt.Errorf("provided movie slug cannot be an empty")
		return 0, wrapErr(ErrValidationFailure, err)
	}

	pageURLString := fmt.Sprintf("%s/movies/%s", &c.config.SiteURL, movieSlug)
	pageURL, _ := url.Parse(pageURLString)
	document, err := c.newDocumentRequestWithContext(ctx, pageURL)
	if err != nil {
		return 0, err
	}

	movieID, err := c.scrapeMovieID(document)
	if err != nil {
		return 0, ErrContentRetrievalFailure
	}

	return movieID, nil
}

func (c *Client) ResolveMovieSlugToID(movieSlug string) (int, error) {
	return c.ResolveMovieSlugToIDWithContext(context.Background(), movieSlug)
}

type TrendingMoviesData struct {
	Movies []SiteMovie `json:"movies"`
}

type TrendingMoviesResponse struct {
	Data TrendingMoviesData `json:"data"`
}

func (c *Client) TrendingMoviesWithContext(ctx context.Context) (
	*TrendingMoviesResponse, error,
) {
	pageURLString := fmt.Sprintf("%s/trending-movies", &c.config.SiteURL)
	pageURL, _ := url.Parse(pageURLString)
	document, err := c.newDocumentRequestWithContext(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	data, err := c.scrapeTrendingMoviesData(document)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	return &TrendingMoviesResponse{*data}, nil
}

func (c *Client) TrendingMovies() (*TrendingMoviesResponse, error) {
	return c.TrendingMoviesWithContext(context.Background())
}

type HomePageContentData struct {
	Popular  []SiteMovie
	Latest   []SiteMovie
	Upcoming []SiteUpcomingMovie
}

type HomePageContentResponse struct {
	Data HomePageContentData `json:"data"`
}

func (c *Client) HomePageContentWithContext(ctx context.Context) (
	*HomePageContentResponse, error,
) {
	document, err := c.newDocumentRequestWithContext(ctx, &c.config.SiteURL)
	if err != nil {
		return nil, err
	}

	data, err := c.scrapeHomePageContentData(document)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	return &HomePageContentResponse{*data}, nil
}

func (c *Client) HomePageContent() (*HomePageContentResponse, error) {
	return c.HomePageContentWithContext(context.Background())
}

type MovieDirectorData struct {
	Director SiteMovieDirector `json:"director"`
}

type MovieDirectorResponse struct {
	Data MovieDirectorData `json:"data"`
}

func (c *Client) GetMovieDirector(movieSlug string) (*MovieDirectorResponse, error) {
	return c.GetMovieDirectorWithContext(context.Background(), movieSlug)
}

func (c *Client) GetMovieDirectorWithContext(ctx context.Context, movieSlug string) (
	*MovieDirectorResponse, error,
) {
	if movieSlug == "" {
		err := fmt.Errorf("provided movie slug cannot be an empty")
		return nil, wrapErr(ErrValidationFailure, err)
	}

	pageURLString := fmt.Sprintf("%s/movies/%s", &c.config.SiteURL, movieSlug)
	pageURL, _ := url.Parse(pageURLString)
	document, err := c.newDocumentRequestWithContext(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	data, err := c.scrapeMovieDirectorData(document)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	return &MovieDirectorResponse{*data}, nil
}

type MovieReviewsData struct {
	Reviews         []SiteMovieReview `json:"reviews"`
	ReviewsMoreLink string            `json:"reviews_more_link"`
}

type MovieReviewsResponse struct {
	Data MovieReviewsData `json:"data"`
}

func (c *Client) GetMovieReviewsWithContext(ctx context.Context, movieSlug string) (
	*MovieReviewsResponse, error,
) {
	if movieSlug == "" {
		err := fmt.Errorf("provided movie slug cannot be an empty")
		return nil, wrapErr(ErrValidationFailure, err)
	}

	pageURLString := fmt.Sprintf("%s/movies/%s", &c.config.SiteURL, movieSlug)
	pageURL, _ := url.Parse(pageURLString)
	document, err := c.newDocumentRequestWithContext(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	data, err := c.scrapeMovieReviewsData(document)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	return &MovieReviewsResponse{*data}, nil
}

func (c *Client) GetMovieReviews(movieSlug string) (*MovieReviewsResponse, error) {
	return c.GetMovieReviewsWithContext(context.Background(), movieSlug)
}

const movieCommentsPerPage = 30

type MovieCommentsData struct {
	CommentsMore bool               `json:"comments_more"`
	Comments     []SiteMovieComment `json:"comments"`
}

type MovieCommentsResponse struct {
	Data MovieCommentsData `json:"data"`
}

func (c *Client) GetMovieCommentsWithContext(ctx context.Context, movieSlug string, page int) (
	*MovieCommentsResponse, error,
) {
	if movieSlug == "" {
		err := fmt.Errorf("provided movie slug cannot be an empty")
		return nil, wrapErr(ErrValidationFailure, err)
	}

	if page < 1 {
		err := fmt.Errorf("provided comment page must be at least 1")
		return nil, wrapErr(ErrValidationFailure, err)
	}

	pageURLString := fmt.Sprintf("%s/movies/%s", &c.config.SiteURL, movieSlug)
	pageURL, _ := url.Parse(pageURLString)
	pageDoc, err := c.newDocumentRequestWithContext(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	meta, err := c.scrapeMovieCommentsMetaData(pageDoc)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	var (
		offset = (page - 1) * movieCommentsPerPage
		isLast = meta.commmentCount-offset <= movieCommentsPerPage
	)

	commentURLString := c.getCommentsURL(meta.movieID, offset)
	commentURL, _ := url.Parse(commentURLString)
	commentDoc, err := c.newDocumentRequestWithContext(ctx, commentURL)
	if err != nil {
		return nil, err
	}

	comments, err := c.scrapeMovieComments(commentDoc)
	if err != nil {
		return nil, ErrContentRetrievalFailure
	}

	data := MovieCommentsData{
		CommentsMore: !isLast,
		Comments:     comments,
	}

	return &MovieCommentsResponse{data}, nil
}

func (c *Client) GetMovieComments(movieSlug string, page int) (*MovieCommentsResponse, error) {
	return c.GetMovieCommentsWithContext(context.Background(), movieSlug, page)
}

type MovieAdditionalDetailsData struct {
	Director SiteMovieDirector  `json:"director"`
	Comments []SiteMovieComment `json:"comments"`
	Reviews  []SiteMovieReview  `json:"reviews"`
}

type MovieAdditionalDetailsResponse struct {
	Data MovieAdditionalDetailsData `json:"data"`
}

func (c *Client) GetMovieAdditionalDetailsWithContext(ctx context.Context, movieSlug string) (
	*MovieAdditionalDetailsResponse, error,
) {
	if movieSlug == "" {
		err := fmt.Errorf("provided movie slug cannot be an empty")
		return nil, wrapErr(ErrValidationFailure, err)
	}

	pageURLString := fmt.Sprintf("%s/movies/%s", &c.config.SiteURL, movieSlug)
	pageURL, _ := url.Parse(pageURLString)
	pageDocument, err := c.newDocumentRequestWithContext(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	var (
		dData, dErr = c.scrapeMovieDirectorData(pageDocument)
		rData, rErr = c.scrapeMovieReviewsData(pageDocument)
		cData, mErr = c.scrapeMovieCommentsMetaData(pageDocument)
	)

	if v := errors.Join(dErr, rErr, mErr); v != nil {
		debug.Panicln(v)
		return nil, ErrContentRetrievalFailure
	}

	commentURLString := c.getCommentsURL(cData.movieID, 0)
	commentURL, _ := url.Parse(commentURLString)
	commentDoc, err := c.newDocumentRequestWithContext(ctx, commentURL)
	if err != nil {
		return nil, err
	}

	comments, cErr := c.scrapeMovieComments(commentDoc)
	if cErr != nil {
		return nil, ErrContentRetrievalFailure
	}

	data := MovieAdditionalDetailsData{
		Comments: comments,
		Director: dData.Director,
		Reviews:  rData.Reviews,
	}

	return &MovieAdditionalDetailsResponse{data}, nil
}

func (c *Client) GetMovieAdditionalDetails(movieSlug string) (*MovieAdditionalDetailsResponse, error) {
	return c.GetMovieAdditionalDetailsWithContext(context.Background(), movieSlug)
}

type TorrentMagnets map[Quality]string

func (c *Client) MagnetLinks(t TorrentInfoGetter) TorrentMagnets {
	var trackers = url.Values{}
	for _, tracker := range c.config.TorrentTrackers {
		trackers.Add("tr", tracker)
	}

	getMagnetFor := func(torrent Torrent) string {
		torrentName := fmt.Sprintf(
			"%s+[%s]+[%s]",
			t.GetTorrentInfo().MovieTitle,
			torrent.Quality,
			strings.ToUpper(c.config.SiteURL.Host),
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
