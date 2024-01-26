package yts

import (
	"errors"
	"fmt"
	"testing"
)

func TestSearchMoviesGetQueryString(t *testing.T) {
	t.Run("returns correct querystring for default filters", func(t *testing.T) {
		defaultFilters := DefaultSearchMoviesFilter()
		received, _ := defaultFilters.getQueryString()
		expected := "genre=All&limit=20&order_by=desc&page=1&quality=All&sort_by=date_added"
		if received != expected {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})

	t.Run("returns joined StructValidation errors when field validations fail", func(t *testing.T) {
		emptyFilters := &SearchMoviesFilters{}
		valErrors := []error{
			&StructValidationError{
				Struct:   "SearchMoviesFilters",
				Field:    "Limit",
				Tag:      "min",
				Value:    0,
				Expected: "1",
			},
			&StructValidationError{
				Struct:   "SearchMoviesFilters",
				Field:    "Page",
				Tag:      "min",
				Value:    0,
				Expected: "1",
			},
			&StructValidationError{
				Struct:   "SearchMoviesFilters",
				Field:    "Quality",
				Tag:      "oneof",
				Value:    "",
				Expected: "All 480p 720p 1080p 1080p.x265 2160p 3D",
			},
			&StructValidationError{
				Struct:   "SearchMoviesFilters",
				Field:    "Genre",
				Tag:      "oneof",
				Value:    "",
				Expected: "All Action Adventure Animation Biography Comedy Crime Documentary Drama Family Fantasy Film-Noir Game-Show History Horror Music Musical Mystery News Reality-TV Romance Sci-Fi Sport Talk-Show Thriller War Western",
			},
			&StructValidationError{
				Struct:   "SearchMoviesFilters",
				Field:    "SortBy",
				Tag:      "oneof",
				Value:    "",
				Expected: "title year rating peers seeds download_count like_count date_added",
			},
			&StructValidationError{
				Struct:   "SearchMoviesFilters",
				Field:    "OrderBy",
				Tag:      "oneof",
				Value:    "",
				Expected: "asc desc",
			},
		}
		_, received := emptyFilters.getQueryString()
		expected := errors.Join(valErrors...)
		if received == nil || received.Error() != expected.Error() {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})
}

func TestMovieDetailsGetQueryString(t *testing.T) {
	t.Run("returns correct querystring for default filters", func(t *testing.T) {
		movieID := 1
		defaultFilters := DefaultMovieDetailsFilters(movieID)
		received, _ := defaultFilters.getQueryString()
		expected := fmt.Sprintf("movie_id=%d&with_cast=true&with_images=true", movieID)
		if received != expected {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})

	t.Run("returns correct querystring with 0 value filters", func(t *testing.T) {
		movieID := 1
		defaultFilters := MovieDetailsFilters{MovieID: 1, WithImages: true}
		received, _ := defaultFilters.getQueryString()
		expected := fmt.Sprintf("movie_id=%d&with_images=true", movieID)
		if received != expected {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})
}
