package yts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	APIBaseURL = "https://yts.mx/api/v2"
	SiteURL    = "https://yts.mx"
)

type Client struct {
	baseURL   string
	siteURL   string
	netClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	if timeout < time.Second*5 || time.Minute*5 < timeout {
		panic(errors.New("YTS client timeout must be between 5 and 300 seconds inclusive"))
	}

	return &Client{
		APIBaseURL,
		SiteURL,
		&http.Client{Timeout: timeout},
	}
}

func (c Client) SearchMovies(ctx context.Context, filters *SearchMoviesFilters) (
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

func (c Client) GetMovieDetails(ctx context.Context, filters *MovieDetailsFilters) (
	*MovieDetailsResponse, error,
) {
	queryString, err := filters.getQueryString()
	if err != nil {
		return nil, err
	}

	parsedPayload := &MovieDetailsResponse{}
	targetURL := c.getEndpointURL("movie_details.json", queryString)
	err = c.getPayloadJSON(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c Client) GetMovieSuggestions(ctx context.Context, movieID int) (
	*MovieSuggestionsResponse, error,
) {
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
	err := c.getPayloadJSON(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c Client) GetTrendingMovies(ctx context.Context) (
	*TrendingMoviesResponse, error,
) {
	var rawPayload []byte
	pageURL := fmt.Sprintf("%s/trending-movies", c.siteURL)
	rawPayload, err := c.getPayloadRaw(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	reader := strings.NewReader(string(rawPayload))
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	selection := document.Find("div.browse-movie-wrap")
	if selection.Length() == 0 {
		return nil, errors.New("no selections found for trending movies")
	}

	trendingMovies := make([]ScrapedMovie, 0)
	selection.Each(func(i int, s *goquery.Selection) {
		trendingMovie := c.parseScrapedMovie(s)
		trendingMovies = append(trendingMovies, trendingMovie)
	})

	response := &TrendingMoviesResponse{
		Data: TrendingMoviesData{trendingMovies},
	}

	return response, nil
}

func (c Client) GetHomePageContent(ctx context.Context) (
	*HomePageContentResponse, error,
) {
	var rawPayload []byte
	rawPayload, err := c.getPayloadRaw(ctx, c.siteURL)
	if err != nil {
		return nil, err
	}

	reader := strings.NewReader(string(rawPayload))
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	const (
		popularCSS  = "div#popular-downloads div.browse-movie-wrap"
		latestCSS   = "div.content-dark div.home-movies div.browse-movie-wrap"
		upcomingCSS = "div.content-dark ~ div.home-content div.browse-movie-wrap"
	)

	var (
		popDownloadSel   = document.Find(popularCSS)
		latestTorrentSel = document.Find(latestCSS)
		upcomingMovieSel = document.Find(upcomingCSS)
	)

	if popDownloadSel.Length() == 0 {
		return nil, errors.New("no elements found for popular movies selection")
	}

	if latestTorrentSel.Length() == 0 {
		return nil, errors.New("no elements found for latest torrents selection")
	}

	if upcomingMovieSel.Length() == 0 {
		return nil, errors.New("no elements found for upcoming movies selection")
	}

	var (
		popDownloads   = make([]ScrapedMovie, 0)
		latestTorrents = make([]ScrapedMovie, 0)
		upcomingMovies = make([]ScrapedUpcomingMovie, 0)
	)

	popDownloadSel.Each(func(i int, s *goquery.Selection) {
		popDownload := c.parseScrapedMovie(s)
		popDownloads = append(popDownloads, popDownload)
	})

	latestTorrentSel.Each(func(i int, s *goquery.Selection) {
		latestTorrent := c.parseScrapedMovie(s)
		latestTorrents = append(latestTorrents, latestTorrent)
	})

	upcomingMovieSel.Each(func(i int, s *goquery.Selection) {
		progressSel := s.Find("div.browse-movie-year progress")
		progress, _ := progressSel.Attr("value")
		progressInt, _ := strconv.Atoi(progress)
		upcomingMovies = append(
			upcomingMovies,
			ScrapedUpcomingMovie{
				ScrapedMovie: c.parseScrapedMovie(s),
				Progress:     progressInt,
			},
		)
	})

	response := &HomePageContentResponse{
		Data: HomePageContentData{
			popDownloads,
			latestTorrents,
			upcomingMovies,
		},
	}

	return response, nil
}

func (c Client) parseScrapedMovie(s *goquery.Selection) ScrapedMovie {
	var (
		bottom   = s.Find("div.browse-movie-bottom")
		anchor   = s.Find("a.browse-movie-link")
		year     = bottom.Find("div.browse-movie-year").Text()
		link, _  = anchor.Attr("href")
		image, _ = anchor.Find("img").Attr("src")
	)

	yearInt, _ := strconv.Atoi(year)
	return ScrapedMovie{
		Title:  bottom.Find("a.browse-movie-title").Text(),
		Year:   yearInt,
		Link:   link,
		Image:  image,
		Rating: anchor.Find("h4.rating").Text(),
	}
}

func (c Client) getPayloadJSON(
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

func (c Client) getPayloadRaw(ctx context.Context, targetURL string) (
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

	response, err := c.netClient.Do(request)
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

func (c Client) getEndpointURL(path, query string) string {
	targetURL := fmt.Sprintf("%s/%s", c.baseURL, path)
	if query == "" {
		return targetURL
	}

	return fmt.Sprintf("%s?%s", targetURL, query)
}
