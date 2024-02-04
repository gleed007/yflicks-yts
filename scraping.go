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

type SiteMovieBase struct {
	Title  string  `json:"title"`
	Year   int     `json:"year"`
	Link   string  `json:"link"`
	Image  string  `json:"image"`
	Genres []Genre `json:"genres"`
}

func (smb *SiteMovieBase) validateScraping() error {
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
		vErr := validation.Validate(genre, validateGenreRule)
		if vErr != nil {
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

func (smb *SiteMovieBase) scrape(s *goquery.Selection) error {
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

	siteMovieBase := SiteMovieBase{
		Title:  bottom.Find(movieTitleCSS).Text(),
		Year:   yearInt,
		Link:   link,
		Image:  image,
		Genres: genres,
	}

	if err := siteMovieBase.validateScraping(); err != nil {
		return wrapErr(ErrSiteScrapingFailure, err)
	}

	*smb = siteMovieBase
	return nil
}

type SiteMovie struct {
	SiteMovieBase
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

func (sm *SiteMovie) validateScraping() error {
	bErr := sm.SiteMovieBase.validateScraping()
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

func (sm *SiteMovie) scrape(s *goquery.Selection) error {
	var (
		anchor = s.Find(movieLinkCSS)
		rating = anchor.Find("h4.rating").Text()
	)

	siteMovie := SiteMovie{}
	siteMovie.Rating = rating
	_ = siteMovie.SiteMovieBase.scrape(s)
	if err := siteMovie.validateScraping(); err != nil {
		return wrapErr(ErrSiteScrapingFailure, err)
	}

	*sm = siteMovie
	return nil
}

type SiteUpcomingMovie struct {
	SiteMovieBase
	Progress int     `json:"progress"`
	Quality  Quality `json:"quality"`
}

func (sum *SiteUpcomingMovie) validateScraping() error {
	const maxProgress = 100
	bErr := sum.SiteMovieBase.validateScraping()
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

func (sum *SiteUpcomingMovie) scrape(s *goquery.Selection) error {
	const expectedYearElemLen = 2

	var (
		yearSel        = s.Find(movieYearCSS)
		progressSel    = yearSel.Find(movieProgressCSS)
		progress, _    = progressSel.Attr("value")
		progressInt, _ = strconv.Atoi(progress)
	)

	var quality Quality
	var yearText = strings.Fields(yearSel.Text())
	if len(yearText) >= expectedYearElemLen {
		quality = Quality(yearText[1])
	}

	upcomingMovie := SiteUpcomingMovie{}
	upcomingMovie.Progress = progressInt
	upcomingMovie.Quality = quality
	_ = upcomingMovie.SiteMovieBase.scrape(s)
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
		trendingMovies = make([]SiteMovie, 0)
		scrapingErrs   = make([]error, 0)
	)

	selection.Each(func(i int, s *goquery.Selection) {
		siteMovie := SiteMovie{}
		err := siteMovie.scrape(s)
		trendingMovies = append(trendingMovies, siteMovie)
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
		popDownloads   = make([]SiteMovie, 0)
		latestTorrents = make([]SiteMovie, 0)
		upcomingMovies = make([]SiteUpcomingMovie, 0)
		scrapingErrs   = make([]error, 0)
	)

	popDownloadSel.Each(func(i int, s *goquery.Selection) {
		siteMovie := SiteMovie{}
		err := siteMovie.scrape(s)
		popDownloads = append(popDownloads, siteMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	latestTorrentSel.Each(func(i int, s *goquery.Selection) {
		siteMovie := SiteMovie{}
		err := siteMovie.scrape(s)
		latestTorrents = append(latestTorrents, siteMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	upcomingMovieSel.Each(func(i int, s *goquery.Selection) {
		upcomingMovie := SiteUpcomingMovie{}
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
