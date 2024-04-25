package yts_test

import (
	"testing"

	yts "github.com/atifcppprogrammer/yflicks-yts"
)

func TestDefaultSearchMoviesFilters(t *testing.T) {
	const queryTerm = "Oppenheimer (2023)"
	got := yts.DefaultSearchMoviesFilters(queryTerm)
	want := &yts.SearchMoviesFilters{
		Limit:         20,
		Page:          1,
		Quality:       yts.QualityAll,
		MinimumRating: 0,
		QueryTerm:     queryTerm,
		Genre:         yts.GenreAll,
		SortBy:        yts.SortByDateAdded,
		OrderBy:       yts.OrderByDesc,
		WithRTRatings: false,
	}

	assertEqual(t, "DefaultSearchMoviesFilter", got, want)
}

func TestDefaultMovieDetailsFilters(t *testing.T) {
	got := yts.DefaultMovieDetailsFilters()
	want := &yts.MovieDetailsFilters{
		WithImages: true,
		WithCast:   true,
	}

	assertEqual(t, "DefaultMovieDetalsFilters", got, want)
}
