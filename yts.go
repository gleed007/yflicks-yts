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
	// The value of the APIBaseURL field for the ClientConfig instance returned by
	// the DefaultClientConfig() function.
	DefaultAPIBaseURL = "https://yts.mx/api/v2"

	// The value of the SiteURL field for the ClientConfig instance returned by the
	// DefaultClientConfig() function.
	DefaultSiteURL = "https://yts.mx"
)

const (
	// TimeoutLimitUpper represents the maximum request Timeout that can be set for
	// the internal *http.Client used by the yts.Client methods.
	TimeoutLimitUpper = 5 * time.Minute

	// TimeoutLimitLower represents the minimum request Timeout that can be set for
	// the internal *http.Client used by the yts.Client methods.
	TimeoutLimitLower = 5 * time.Second
)

var debug = newLogger()

// A ClientConfig allows you to configure the behavior of the `yts.Client` instance
// created by NewClient() function.
type ClientConfig struct {
	// The base URL for the YTS API used by *yts.Client methods, you will likely never
	// have to specify a value for this other than DefaultAPIBaseURL
	APIBaseURL url.URL

	// The base URL for the YTS website used by *yts.Client methods, you will likely
	// never have to specify a value for this other than DefaultSiteURL
	SiteURL url.URL

	// The list of torrent tracker URLs used by the `MagnetLinks()` method for
	// preparing magnet links for movie torrents.
	TorrentTrackers []string

	// The timeout duration after which a http request will be cancelled by a client
	// method, this value is passed to the internal *http.Client instance used by the
	// *yts.Client.
	RequestTimeout time.Duration

	// This flag "switches on" an internal logger and is intended for use by developers
	// for debugging purposes, if you encounter a bug in this package turning this flag
	// on will reveal greater detail regarding the error in question.
	Debug bool
}

// A Client represents the main struct type provided by the `yts` package, you use
// this instance's method to interact with the YTS API and fetch content scraped
// from the YTS website.
type Client struct {
	config    ClientConfig
	netClient *http.Client
}

var (
	// ErrInvalidClientConfig indicates that you attempted to create a `yts.Client`
	// instance with an invalid client config, the error description will carry
	// further details.
	ErrInvalidClientConfig = errors.New("invalid_client_config")

	// ErrContentRetrievalFailure indicates that a yts.Client method failed to scrape
	// content from the YTS Site, and may indicate the presence of a bug, please
	// report these by creating an issue at the following URL.
	// https://github.com/atifcppprogrammer/yflicks-yts/issues/new.
	ErrContentRetrievalFailure = errors.New("content_retrieval_failure")

	// ErrFilterValidationFailure is reported when you provided invalid values for an
	// instance of SearchMoviesFilters, the error description will carry further
	// details.
	ErrFilterValidationFailure = errors.New("filter_validation_failure")

	// A ErrValidationFailure is reported whenever you provided an invalid value for
	// an input argument to one of the methods of the yts.Client method, the error
	// description will carry further information.
	ErrValidationFailure = errors.New("validation_failure")
)

func wrapErr(sentinel error, others ...error) error {
	return fmt.Errorf("%w: %s", sentinel, errors.Join(others...))
}

// DefaultTorrentTrackers returns the list of torrent trackers which are used by
// the default client configuration i.e. the ClientConfig instance, return by the
// DefaultClientConfig() function. They are used for generating the magnets links
// for YTS torrents.
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

// DefaultClientConfig returns a `ClientConfig` instance, with sensible default
// field values.
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

// NewClientConfig creates `*Client` instance for the provided client config, an
// error will be returned in the event the provided config is invalid.
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

// NewClient returns a new `*yts.Client` instance with the internal ClientConfig
// being the one returned by the `DefaultClientConfig()` function.
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

// A BaseResponse instance helps to model the response type of the following
// endpoints the YTS API.
//
// - "/api/v2/list_movies.json"
// - "/api/v2/movie_details.json"
// - "/api/v2/movie_suggestions.json"
type BaseResponse struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Meta          `json:"@meta"`
}

// A SearchMoviesResponse models the response of the "/api/v2/list_movies.json"
// endpoint of YTS API (https://yts.mx/api#list_movies).
type SearchMoviesResponse struct {
	BaseResponse
	Data SearchMoviesData `json:"data"`
}

// SearchMoviesWithContext is the same as the SearchMovies method but requires a
// context.Context argument to be passed, this context is then passed to the
// http.NewRequestWithContext call used for making the network request.
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

// SearchMovies returns the response of the "/api/v2/list_movies.json" endpoint with
// the provided search filters, the provided filter values are validated internally
// and an error is returned in the event validation fails.
func (c *Client) SearchMovies(filters *SearchMoviesFilters) (*SearchMoviesResponse, error) {
	return c.SearchMoviesWithContext(context.Background(), filters)
}

type MovieDetailsData struct {
	Movie MovieDetails `json:"movie"`
}

// A MovieDetailsResponse models the response of the "/api/v2/movie_details.json"
// endpoint of YTS API (https://yts.mx/api#movie_details).
type MovieDetailsResponse struct {
	BaseResponse
	Data MovieDetailsData `json:"data"`
}

// MovieDetailsWithContext is the same as the MovieDetails method but requires
// a context.Context argument to be passed, this context is then passed to the
// http.NewRequestWithContext call used for making the network request.
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

// MovieDetails returns the response of "/api/v2/movie_details.json" endpoint
// with the provided filters and movieID, the provided movieID must be positive
// integer a 404 error will be returned if no movie is found for provided movieID.
func (c *Client) MovieDetails(movieID int, filters *MovieDetailsFilters) (*MovieDetailsResponse, error) {
	return c.MovieDetailsWithContext(context.Background(), movieID, filters)
}

type MovieSuggestionsData struct {
	MovieCount int     `json:"movie_count"`
	Movies     []Movie `json:"movies"`
}

// A MovieSuggestionsResponse models the response of the "/api/v2/movie_suggestions.json"
// endpoint of YTS API (https://yts.mx/api#movie_suggestions).
type MovieSuggestionsResponse struct {
	BaseResponse
	Data MovieSuggestionsData `json:"data"`
}

// MovieSuggestionsWithContext is the same as the MovieSuggestions method but
// requires a context.Context argument to be passed, this context is then passed to
// the http.NewRequestWithContext call used for making the network request.
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

// MovieSuggestions returns the response of the "/api/v2/movie_suggestions.json"
// endpoint, the provided movieID must be positive integer, a 404 error will be
// returned if no movie is found for provided movieID.
func (c *Client) MovieSuggestions(movieID int) (*MovieSuggestionsResponse, error) {
	return c.MovieSuggestionsWithContext(context.Background(), movieID)
}

type TrendingMoviesData struct {
	Movies []SiteMovie `json:"movies"`
}

// A TrendingMoviesResponse holds the content retrieved by scraping the /trending
// page of the YTS website, the content in question being the movies currently
// trending in the past 24 Hours.
type TrendingMoviesResponse struct {
	Data TrendingMoviesData `json:"data"`
}

// TrendingMoviesWithContext is the same as the TrendingMovies method but
// requires a context.Context argument to be passed, this context is then passed to
// the http.NewRequestWithContext call used for making the network request.
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

// TrendingMovies method scrapes the "/trending" page of the YTS website and
// returns the movies shown therein as an instance of *TrendingMoviesResponse
func (c *Client) TrendingMovies() (*TrendingMoviesResponse, error) {
	return c.TrendingMoviesWithContext(context.Background())
}

type HomePageContentData struct {
	Popular  []SiteMovie         `json:"popular"`
	Latest   []SiteMovie         `json:"latest"`
	Upcoming []SiteUpcomingMovie `json:"upcoming"`
}

// A HomePageContentResponse holds the content retrieved by scraping the /trending
// page of the YTS website, the content in question being the current popular,
// trending and upcoming movie torrents.
type HomePageContentResponse struct {
	Data HomePageContentData `json:"data"`
}

// HomePageContentWithContext is the same as the HomePageContent method but
// requires a context.Context argument to be passed, this context is then passed to
// the http.NewRequestWithContext call used for making the network request.
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

// HomePageContent method scrapes the popular, latest torrents and upcoming
// movies sections of the YTS website's "/" home page and returns this as an
// instance of *HomePageContentResponse.
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
	pageCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	pageDoc, err := c.newDocumentRequestWithContext(pageCtx, pageURL)
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
	commentCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	commentDoc, err := c.newDocumentRequestWithContext(commentCtx, commentURL)
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
	pageCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	pageDocument, err := c.newDocumentRequestWithContext(pageCtx, pageURL)
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
	commentCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	commentDoc, err := c.newDocumentRequestWithContext(commentCtx, commentURL)
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

// A TorrentMagnets is the return type of MagnetLinks method of a `yts.Client`
type TorrentMagnets map[Quality]string

// MagnetLinks returns a TorrentMagnets instance for all torrents returned by
// the provided TorrentInfoGetter instance, you can pass instances of Movie and
// MoviePartial into this method directly since they both implement the
// TorrentInfoGetter interface.
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
