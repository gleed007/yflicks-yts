package yts

import (
	"fmt"
	"net/url"
)

type SearchMoviesFilters struct {
	Limit         int    `json:"limit"           validate:"min=1,max=50"`
	Page          int    `json:"page"            validate:"min=1"`
	Quality       string `json:"quality"         validate:"oneof=all 480p 720p 1080p 1080p.x265 2160p 3D"`
	MinimumRating int    `json:"minimum_rating"  validate:"min=0,max=9"`
	QueryTerm     string `json:"query_term"`
	Genre         string `json:"genre"           validate:"oneof=all action adventure animation biography comedy crime documentary drama family fantasy film-noir game-show history horror music musical mystery news reality-tv romance sci-fi sport talk-show thriller war western"`
	SortBy        string `json:"sort_by"         validate:"oneof=title year rating peers seeds download_count like_count date_added"`
	OrderBy       string `json:"order_by"        validate:"oneof=asc desc"`
	WithRTRatings bool   `json:"with_rt_ratings" validate:"boolean"`
}

type MovieDetailsFilters struct {
	MovieID    int  `json:"movie_id"    validate:"required,min=1"`
	WithImages bool `json:"with_images" validate:"boolean"`
	WithCast   bool `json:"with_cast"   validate:"boolean"`
}

func DefaultSearchMoviesFilter() *SearchMoviesFilters {
	const (
		defaultPageLimit     = 20
		defaultMinimumRating = 0
	)

	return &SearchMoviesFilters{
		Limit:         defaultPageLimit,
		Page:          1,
		Quality:       "all",
		MinimumRating: 0,
		QueryTerm:     "",
		Genre:         "all",
		SortBy:        "date_added",
		OrderBy:       "desc",
		WithRTRatings: false,
	}
}

func DefaultMovieDetailsFilters(movieID int) *MovieDetailsFilters {
	return &MovieDetailsFilters{
		MovieID:    movieID,
		WithImages: true,
		WithCast:   true,
	}
}

func (f *SearchMoviesFilters) getQueryString() (string, error) {
	if err := validateStruct("SearchMoviesFilters", f); err != nil {
		return "", err
	}

	var (
		queryValues  = url.Values{}
		queryMapping = map[string]interface{}{
			"limit":           f.Limit,
			"page":            f.Page,
			"quality":         f.Quality,
			"minimum_rating":  f.MinimumRating,
			"query_term":      f.QueryTerm,
			"genre":           f.Genre,
			"sort_by":         f.SortBy,
			"order_by":        f.OrderBy,
			"with_rt_ratings": f.WithRTRatings,
		}
	)

	for query, value := range queryMapping {
		switch v := value.(type) {
		case int:
			if v != 0 {
				queryValues.Add(query, fmt.Sprintf("%d", v))
			}
		case bool:
			if v {
				queryValues.Add(query, "true")
			}
		case string:
			if v != "" {
				queryValues.Add(query, v)
			}
		}
	}

	return queryValues.Encode(), nil
}

func (f *MovieDetailsFilters) getQueryString() (string, error) {
	if err := validateStruct("MovieDetailsFilters", f); err != nil {
		return "", err
	}

	queryValues := url.Values{}
	queryValues.Add("movie_id", fmt.Sprintf("%d", f.MovieID))
	if f.WithImages {
		queryValues.Add("with_images", "true")
	}

	if f.WithCast {
		queryValues.Add("with_cast", "true")
	}

	return queryValues.Encode(), nil
}