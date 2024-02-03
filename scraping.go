package yts

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

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
	movieGenreCSS    = "div.browse-movie-wrap h4:not([class='rating'])"
)

type ScrapedMovieBase struct {
	Title  string  `json:"title"`
	Year   int     `json:"year"`
	Link   string  `json:"link"`
	Image  string  `json:"image"`
	Genres []Genre `json:"genres"`
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

	var genreErrs error
	for i, genre := range smb.Genres {
		err := validation.Validate(&genre, validateGenreRule)
		if err != nil {
			genreErrs = errors.Join(
				genreErrs,
				fmt.Errorf("genres: invalid genres[%d] = %q", i, genre),
			)
		}
	}

	if err == nil && genreErrs == nil {
		return nil
	}

	return wrapErr(ErrValidationFailure, err, genreErrs)
}

func (smb *ScrapedMovieBase) scrape(s *goquery.Selection) error {
	var (
		bottom   = s.Find(movieBottomCSS)
		anchor   = s.Find(movieLinkCSS)
		year     = bottom.Find(movieYearCSS).Text()
		genreSel = s.Find(movieGenreCSS)
		link, _  = anchor.Attr("href")
		image, _ = anchor.Find("img").Attr("src")
	)

	var yearInt int
	var yearText = strings.Fields(year)
	if len(yearText) >= 1 {
		yearInt, _ = strconv.Atoi(yearText[0])
	}

	genres := make([]Genre, 0)
	genreSel.Each(func(i int, s *goquery.Selection) {
		genres = append(genres, Genre(s.Text()))
	})

	scrapedMovieBase := ScrapedMovieBase{
		Title:  bottom.Find(movieTitleCSS).Text(),
		Year:   yearInt,
		Link:   link,
		Image:  image,
		Genres: genres,
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

var validateRatingRule = validation.NewStringRule(
	func(input string) bool {
		pattern := `^\d+(\.\d+)?\s*\/\s*10(\.\d+)?$`
		re := regexp.MustCompile(pattern)
		return re.MatchString(input)
	},
	`expecting rating in "[0-9].[0-9] / 10" format`,
)

func (sm *ScrapedMovie) validateScraping() error {
	bErr := sm.ScrapedMovieBase.validateScraping()
	mErr := validation.ValidateStruct(
		sm,
		validation.Field(
			&sm.Rating,
			validateRatingRule,
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
	Progress int     `json:"progress"`
	Quality  Quality `json:"quality"`
}

func (sum *ScrapedUpcomingMovie) validateScraping() error {
	const maxProgress = 100
	bErr := sum.ScrapedMovieBase.validateScraping()
	mErr := validation.ValidateStruct(
		sum,
		validation.Field(
			&sum.Progress,
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
		yearSel        = s.Find(movieYearCSS)
		progressSel    = yearSel.Find(movieProgressCSS)
		progress, _    = progressSel.Attr("value")
		progressInt, _ = strconv.Atoi(progress)
	)

	var quality Quality
	var yearText = strings.Fields(yearSel.Text())
	if len(yearText) >= 2 {
		quality = Quality(yearText[1])
	}

	upcomingMovie := ScrapedUpcomingMovie{}
	upcomingMovie.Progress = progressInt
	upcomingMovie.Quality = quality
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
