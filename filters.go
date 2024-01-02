package yts

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

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

func (f *SearchMoviesFilters) validateFields() error {
	err := validate.Struct(f)
	if err == nil {
		return nil
	}

	filterErrors := make([]error, 0)
	for _, err := range err.(validator.ValidationErrors) {
		filterError := &FilterValidationError{
			filter:   "SearchMovieFilters",
			field:    err.Field(),
			tag:      err.ActualTag(),
			value:    err.Value(),
			expected: err.Param(),
		}
		filterErrors = append(filterErrors, filterError)
	}

	return errors.Join(filterErrors...)
}

func (f *SearchMoviesFilters) getQueryString() (string, error) {
	if err := f.validateFields(); err != nil {
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
