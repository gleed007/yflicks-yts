package yts

import (
	"fmt"
	"net/url"
)

type SearchMoviesFilters struct {
	Limit         int    `json:"limit"`
	Page          int    `json:"page"`
	Quality       string `json:"quality"`
	MinimumRating int    `json:"minimum_rating"`
	QueryTerm     string `json:"query_term"`
	Genre         string `json:"genre"`
	SortBy        string `json:"sort_by"`
	OrderBy       string `json:"order_by"`
	WithRTRatings bool   `json:"with_rt_ratings"`
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

func (f *SearchMoviesFilters) getQueryString() string {
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

	return queryValues.Encode()
}
