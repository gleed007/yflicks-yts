package yts

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	trendingCSS = "div.browse-movie-wrap"
	popularCSS  = "div#popular-downloads div.browse-movie-wrap"
	latestCSS   = "div.content-dark div.home-movies div.browse-movie-wrap"
	upcomingCSS = "div.content-dark ~ div.home-content div.browse-movie-wrap"
)

const (
	movieBottomCSS   = "div.browse-movie-bottom"
	movieLinkCSS     = "a.browse-movie-link"
	movieYearCSS     = "div.browse-movie-year"
	movieTitleCSS    = "a.browse-movie-title"
	movieProgressCSS = "div.browse-movie-year progress"
)

type ScrapedMovieBase struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Link  string `json:"link"`
	Image string `json:"image"`
}

func (smb *ScrapedMovieBase) validateScraping() error {
	err := validation.ValidateStruct(
		smb,
		validation.Field(
			&smb.Title,
			validation.Required,
		),
		validation.Field(
			&smb.Year,
			validation.Required,
		),
		validation.Field(
			&smb.Link,
			validation.Required,
			is.URL,
		),
		validation.Field(
			&smb.Link,
			validation.Required,
		),
	)

	if err == nil {
		return nil
	}

	return wrapErr(ErrValidationFailure, err)
}

func (smb *ScrapedMovieBase) scrape(s *goquery.Selection) error {
	var (
		bottom   = s.Find(movieBottomCSS)
		anchor   = s.Find(movieLinkCSS)
		year     = bottom.Find(movieYearCSS).Text()
		link, _  = anchor.Attr("href")
		image, _ = anchor.Find("img").Attr("src")
	)

	yearInt, _ := strconv.Atoi(year)
	scrapedMovieBase := ScrapedMovieBase{
		Title: bottom.Find(movieTitleCSS).Text(),
		Year:  yearInt,
		Link:  link,
		Image: image,
	}

	if err := scrapedMovieBase.validateScraping(); err != nil {
		return wrapErr(ErrSiteScrapingFailure, err)
	}

	*smb = scrapedMovieBase
	return nil
}

type ScrapedMovie struct {
	ScrapedMovieBase
	Rating string `json:"rating"`
}

func (sm *ScrapedMovie) validateScraping() error {
	bErr := sm.ScrapedMovieBase.validateScraping()
	mErr := validation.ValidateStruct(
		sm,
		validation.Field(
			&sm.Rating,
			validation.Required,
		),
	)
	if bErr == nil && mErr == nil {
		return nil
	}

	return wrapErr(ErrValidationFailure, bErr, mErr)
}

func (sm *ScrapedMovie) scrape(s *goquery.Selection) error {
	var (
		anchor = s.Find(movieLinkCSS)
		rating = anchor.Find("h4.rating").Text()
	)

	scrapedMovie := ScrapedMovie{}
	scrapedMovie.Rating = rating
	_ = scrapedMovie.ScrapedMovieBase.scrape(s)
	if err := scrapedMovie.validateScraping(); err != nil {
		return wrapErr(ErrSiteScrapingFailure, err)
	}

	*sm = scrapedMovie
	return nil
}

type ScrapedUpcomingMovie struct {
	ScrapedMovieBase
	Progress int `json:"progress"`
}

func (sum *ScrapedUpcomingMovie) validateScraping() error {
	const maxProgress = 100
	bErr := sum.ScrapedMovieBase.validateScraping()
	mErr := validation.ValidateStruct(
		sum,
		validation.Field(
			&sum.Progress,
			validation.Required,
			validation.Min(0),
			validation.Max(maxProgress),
		),
	)
	if bErr == nil && mErr == nil {
		return nil
	}

	return wrapErr(ErrValidationFailure, bErr, mErr)
}

func (sum *ScrapedUpcomingMovie) scrape(s *goquery.Selection) error {
	var (
		progressSel    = s.Find(movieProgressCSS)
		progress, _    = progressSel.Attr("value")
		progressInt, _ = strconv.Atoi(progress)
	)

	upcomingMovie := ScrapedUpcomingMovie{}
	upcomingMovie.Progress = progressInt
	_ = upcomingMovie.ScrapedMovieBase.scrape(s)
	if err := upcomingMovie.validateScraping(); err != nil {
		return wrapErr(ErrSiteScrapingFailure, err)
	}

	*sum = upcomingMovie
	return nil
}

func (c *Client) scrapeTrendingMoviesData(r io.Reader) (*TrendingMoviesData, error) {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	selection := document.Find(trendingCSS)
	if selection.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", trendingCSS)
		return nil, wrapErr(ErrSiteScrapingFailure, err)
	}

	var (
		trendingMovies = make([]ScrapedMovie, 0)
		scrapingErrs   = make([]error, 0)
	)

	selection.Each(func(i int, s *goquery.Selection) {
		scrapedMovie := ScrapedMovie{}
		err := scrapedMovie.scrape(s)
		trendingMovies = append(trendingMovies, scrapedMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	if err := errors.Join(scrapingErrs...); err != nil {
		return nil, err
	}

	return &TrendingMoviesData{trendingMovies}, nil
}

func (c *Client) scrapeHomePageContentData(r io.Reader) (*HomePageContentData, error) {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var (
		popDownloadSel   = document.Find(popularCSS)
		latestTorrentSel = document.Find(latestCSS)
		upcomingMovieSel = document.Find(upcomingCSS)
	)

	if popDownloadSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", popularCSS)
		return nil, wrapErr(ErrSiteScrapingFailure, err)
	}

	if latestTorrentSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", latestCSS)
		return nil, wrapErr(ErrSiteScrapingFailure, err)
	}

	if upcomingMovieSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", upcomingCSS)
		return nil, wrapErr(ErrSiteScrapingFailure, err)
	}

	var (
		popDownloads   = make([]ScrapedMovie, 0)
		latestTorrents = make([]ScrapedMovie, 0)
		upcomingMovies = make([]ScrapedUpcomingMovie, 0)
		scrapingErrs   = make([]error, 0)
	)

	popDownloadSel.Each(func(i int, s *goquery.Selection) {
		scrapedMovie := ScrapedMovie{}
		err := scrapedMovie.scrape(s)
		popDownloads = append(popDownloads, scrapedMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	latestTorrentSel.Each(func(i int, s *goquery.Selection) {
		scrapedMovie := ScrapedMovie{}
		err := scrapedMovie.scrape(s)
		latestTorrents = append(latestTorrents, scrapedMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	upcomingMovieSel.Each(func(i int, s *goquery.Selection) {
		upcomingMovie := ScrapedUpcomingMovie{}
		err := upcomingMovie.scrape(s)
		upcomingMovies = append(upcomingMovies, upcomingMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	if err := errors.Join(scrapingErrs...); err != nil {
		return nil, err
	}

	response := &HomePageContentData{
		Popular:  popDownloads,
		Latest:   latestTorrents,
		Upcoming: upcomingMovies,
	}

	return response, nil
}
