package yts_test

import (
	"reflect"
	"testing"

	yts "github.com/atifcppprogrammer/yflicks-yts"
)

func TestDefaultSearchMoviesFilter(t *testing.T) {
	const queryTerm = "Oppenheimer (2023)"
	got := yts.DefaultSearchMoviesFilter(queryTerm)
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

	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.DefaultSearchMoviesFilter() = %v, want %v", got, want)
	}
}

func TestDefaultMovieDetailsFilters(t *testing.T) {
	got := yts.DefaultMovieDetailsFilters()
	want := &yts.MovieDetailsFilters{
		WithImages: true,
		WithCast:   true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.DefaultMovieDetailsFilters() = %v, want %v", got, want)
	}
}
