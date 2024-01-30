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
	SiteDomain = "yts.mx"
)

type Client struct {
	apiBaseURL      string
	siteURL         string
	siteDomain      string
	netClient       *http.Client
	torrentTrackers []string
}

func NewClient(timeout time.Duration) *Client {
	if timeout < time.Second*5 || time.Minute*5 < timeout {
		panic(errors.New("YTS client timeout must be between 5 and 300 seconds inclusive"))
	}

	return &Client{
		APIBaseURL,
		SiteURL,
		SiteDomain,
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

func (c *Client) GetMovieDetails(ctx context.Context, movieID int, filters *MovieDetailsFilters) (
	*MovieDetailsResponse, error,
) {
	if movieID <= 0 {
		return nil, errors.New("provided movieID must be at least 1")
	}

	queryString, err := filters.getQueryString()
	if err != nil {
		return nil, err
	}

	parsedPayload := &MovieDetailsResponse{}
	queryString = fmt.Sprintf("movie_id=%d&%s", movieID, queryString)
	targetURL := c.getEndpointURL("movie_details.json", queryString)
	err = c.getPayloadJSON(ctx, targetURL, parsedPayload)
	if err != nil {
		return nil, err
	}

	return parsedPayload, nil
}

func (c *Client) GetMovieSuggestions(ctx context.Context, movieID int) (
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

func (c *Client) GetTrendingMovies(ctx context.Context) (
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

func (c *Client) GetHomePageContent(ctx context.Context) (
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
		return "", fmt.Errorf("no torrent found having quality %s", q)
	}

	torrentName := fmt.Sprintf(
		"%s+[%s]+[%s]",
		torrentInfo.MovieTitle, q, strings.ToUpper(c.siteDomain),
	)

	var trackers = url.Values{}
	for _, tracker := range c.torrentTrackers {
		trackers.Add("tr", tracker)
	}

	magnet := fmt.Sprintf(
		"magnet:?xt=urn:btih:%s&dn=%s&%s",
		foundTorrent.Hash, url.QueryEscape(torrentName), trackers.Encode(),
	)

	return magnet, nil
}

func (c *Client) parseScrapedMovie(s *goquery.Selection) ScrapedMovie {
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

func (c *Client) getEndpointURL(path, query string) string {
	targetURL := fmt.Sprintf("%s/%s", c.apiBaseURL, path)
	if query == "" {
		return targetURL
	}

	return fmt.Sprintf("%s?%s", targetURL, query)
}
