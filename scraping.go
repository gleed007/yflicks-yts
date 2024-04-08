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
	directorCSS = "div#movie-content div#movie-sub-info div#crew div.directors"
)

const (
	movieBottomCSS   = "div.browse-movie-bottom"
	movieLinkCSS     = "a.browse-movie-link"
	movieYearCSS     = "div.browse-movie-year"
	movieTitleCSS    = "a.browse-movie-title"
	movieProgressCSS = "div.browse-movie-year progress"
	movieGenreCSS    = "div.browse-movie-wrap h4:not([class='rating'])"
)

const (
	directorThumbCSS = "div.list-cast a.avatar-thumb img"
	directorNameCSS  = "div.list-cast-info a.name-cast span span"
)

const (
	reviewsCSS      = "div#movie-reviews div.review"
	reviewRatingCSS = "div.review-properties span.review-rating"
	reviewAuthorCSS = "div.review-properties span.review-author"
	reviewsMoreCSS  = "div#movie-reviews a.more-reviews"
)

// The SiteMovieBase type contains all the information required by both the
// and SiteMovieUpcoming types.
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
			&smb.Image,
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

	return errors.Join(err, genreErrs)
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
		yearI, err := strconv.Atoi(yearText[0])
		if err != nil {
			return err
		}
		yearInt = yearI
	}

	genres := make([]Genre, 0)
	genreSel.Each(func(_ int, s *goquery.Selection) {
		genres = append(genres, Genre(s.Text()))
	})

	smb.Title = bottom.Find(movieTitleCSS).Text()
	smb.Year = yearInt
	smb.Link = link
	smb.Image = image
	smb.Genres = genres
	return smb.validateScraping()
}

// A SiteMovie instance represents all the information provided for each "movie card"
// show on the following pages of the YTS website.
//
// - The popular and latest movie sections on the home page of the YTS website.
// - The trending movies shown on the YTS website.
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
	return errors.Join(bErr, mErr)
}

func (sm *SiteMovie) scrape(s *goquery.Selection) error {
	var (
		anchor = s.Find(movieLinkCSS)
		rating = anchor.Find("h4.rating").Text()
	)

	sm.Rating = rating
	_ = sm.SiteMovieBase.scrape(s)
	return sm.validateScraping()
}

// A SiteUpcomingMovie instance represents all the information provided for each
// "movie card" on the upcoming section on the home page of the YTS website.
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
			&sum.Quality,
			validateQualityRule,
		),
		validation.Field(
			&sum.Progress,
			validation.Min(0),
			validation.Max(maxProgress),
		),
	)
	return errors.Join(bErr, mErr)
}

func (sum *SiteUpcomingMovie) scrape(s *goquery.Selection) error {
	const expectedYearElemLen = 2

	var (
		yearSel     = s.Find(movieYearCSS)
		progressSel = yearSel.Find(movieProgressCSS)
		progress, _ = progressSel.Attr("value")
	)

	progressInt, err := strconv.Atoi(progress)
	if err != nil {
		return err
	}

	var quality Quality
	var yearText = strings.Fields(yearSel.Text())
	if len(yearText) >= expectedYearElemLen {
		quality = Quality(yearText[1])
	}

	sum.Progress = progressInt
	sum.Quality = quality
	_ = sum.SiteMovieBase.scrape(s)
	return sum.validateScraping()
}

type SiteMovieDirector struct {
	Name          string `json:"name"`
	URLSmallImage string `json:"url_small_image"`
}

func (smd *SiteMovieDirector) validateScraping() error {
	return validation.ValidateStruct(
		smd,
		validation.Field(
			&smd.Name,
			validation.Required,
		),
		validation.Field(
			&smd.URLSmallImage,
			validation.Required,
			is.URL,
		),
	)
}

func (smd *SiteMovieDirector) scrape(s *goquery.Selection) error {
	var (
		nameSel     = s.Find(directorNameCSS)
		thumbImgSel = s.Find(directorThumbCSS)
	)

	smd.Name = nameSel.Text()
	smd.URLSmallImage, _ = thumbImgSel.Attr("src")
	return smd.validateScraping()
}

type SiteMovieReview struct {
	Author  string `json:"author"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Rating  string `json:"rating"`
}

func (smr *SiteMovieReview) validateScraping() error {
	return validation.ValidateStruct(
		smr,
		validation.Field(
			&smr.Author,
			validation.Required,
		),
		validation.Field(
			&smr.Title,
			validation.Required,
		),
		validation.Field(
			&smr.Content,
			validation.Required,
		),
		validation.Field(
			&smr.Rating,
			validateRatingRule,
		),
	)
}

func (smr *SiteMovieReview) scrape(s *goquery.Selection) error {
	var (
		authorSel  = s.Find(reviewAuthorCSS)
		ratingSel  = s.Find(reviewRatingCSS)
		titleSel   = s.Find("h4")
		contentSel = s.Find("article")
	)

	smr.Author = authorSel.Text()
	smr.Rating = ratingSel.Text()
	smr.Title = titleSel.Text()
	smr.Content = contentSel.Text()
	return smr.validateScraping()
}

func (c *Client) scrapeTrendingMoviesData(d *goquery.Document) (*TrendingMoviesData, error) {
	selection := d.Find(trendingCSS)
	if selection.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", trendingCSS)
		debug.Println(err)
		return nil, err
	}

	var (
		trendingMovies = make([]SiteMovie, 0)
		scrapingErrs   = make([]error, 0)
	)

	selection.Each(func(i int, s *goquery.Selection) {
		siteMovie := SiteMovie{}
		err := siteMovie.scrape(s)
		if err != nil {
			err = fmt.Errorf("trending, i=%d, %w", i, err)
		}

		trendingMovies = append(trendingMovies, siteMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	if err := errors.Join(scrapingErrs...); err != nil {
		debug.Println(err)
		return nil, err
	}

	return &TrendingMoviesData{trendingMovies}, nil
}

func (c *Client) scrapeHomePageContentData(d *goquery.Document) (*HomePageContentData, error) {
	var (
		popDownloadSel   = d.Find(popularCSS)
		latestTorrentSel = d.Find(latestCSS)
		upcomingMovieSel = d.Find(upcomingCSS)
	)

	if popDownloadSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", popularCSS)
		debug.Println(err)
		return nil, err
	}

	if latestTorrentSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", latestCSS)
		debug.Println(err)
		return nil, err
	}

	if upcomingMovieSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", upcomingCSS)
		debug.Println(err)
		return nil, err
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
		if err != nil {
			err = fmt.Errorf("popular, i=%d, %w", i, err)
		}

		popDownloads = append(popDownloads, siteMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	latestTorrentSel.Each(func(i int, s *goquery.Selection) {
		siteMovie := SiteMovie{}
		err := siteMovie.scrape(s)
		if err != nil {
			err = fmt.Errorf("latest, i=%d, %w", i, err)
		}

		latestTorrents = append(latestTorrents, siteMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	upcomingMovieSel.Each(func(i int, s *goquery.Selection) {
		upcomingMovie := SiteUpcomingMovie{}
		err := upcomingMovie.scrape(s)
		if err != nil {
			err = fmt.Errorf("upcoming, i=%d, %w", i, err)
		}

		upcomingMovies = append(upcomingMovies, upcomingMovie)
		scrapingErrs = append(scrapingErrs, err)
	})

	if err := errors.Join(scrapingErrs...); err != nil {
		debug.Println(err)
		return nil, err
	}

	response := &HomePageContentData{
		Popular:  popDownloads,
		Latest:   latestTorrents,
		Upcoming: upcomingMovies,
	}

	return response, nil
}

func (c *Client) scrapeMovieDirectorData(r io.Reader) (*MovieDirectorData, error) {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		debug.Println(err)
		return nil, err
	}

	directorSel := document.Find(directorCSS)
	if directorSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", directorCSS)
		debug.Println(err)
		return nil, err
	}

	director := &SiteMovieDirector{}
	if err := director.scrape(directorSel); err != nil {
		debug.Println(err)
		return nil, err
	}

	return &MovieDirectorData{*director}, nil
}

func (c *Client) scrapeMovieReviewsData(r io.Reader) (*MovieReviewsData, error) {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		debug.Println(err)
		return nil, err
	}

	reviewsSel := document.Find(reviewsCSS)
	if reviewsSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", reviewsCSS)
		debug.Println(err)
		return nil, err
	}

	reviewsMoreSel := document.Find(reviewsMoreCSS)
	if reviewsMoreSel.Length() == 0 {
		err := fmt.Errorf("no elements found for %q", reviewsMoreCSS)
		debug.Println(err)
		return nil, err
	}

	reviewsMoreURL, _ := reviewsMoreSel.Attr("href")
	if err := validation.Validate(reviewsMoreURL, is.URL); err != nil {
		err := fmt.Errorf(`invalid "href" found for %q`, reviewsMoreCSS)
		debug.Println(err)
		return nil, err
	}

	var (
		movieReviews = make([]SiteMovieReview, 0)
		scrapingErrs = make([]error, 0)
	)

	reviewsSel.Each(func(i int, s *goquery.Selection) {
		movieReview := SiteMovieReview{}
		err := movieReview.scrape(s)
		if err != nil {
			err = fmt.Errorf("reviews, i=%d, %w", i, err)
		}

		movieReviews = append(movieReviews, movieReview)
		scrapingErrs = append(scrapingErrs, err)
	})

	if err := errors.Join(scrapingErrs...); err != nil {
		debug.Println(err)
		return nil, err
	}

	return &MovieReviewsData{
		Reviews:         movieReviews,
		ReviewsMoreLink: reviewsMoreURL,
	}, nil
}
