package yts

import (
	"errors"
	"fmt"
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Genre string

const (
	GenreAll         Genre = "All"
	GenreAction      Genre = "Action"
	GenreAdventure   Genre = "Adventure"
	GenreAnimation   Genre = "Animation"
	GenreBiography   Genre = "Biography"
	GenreComedy      Genre = "Comedy"
	GenreCrime       Genre = "Crime"
	GenreDocumentary Genre = "Documentary"
	GenreDrama       Genre = "Drama"
	GenreFamily      Genre = "Family"
	GenreFantasy     Genre = "Fantasy"
	GenreFilmNoir    Genre = "Film-Noir"
	GenreGameShow    Genre = "Game-Show"
	GenreHistory     Genre = "History"
	GenreHorror      Genre = "Horror"
	GenreMusic       Genre = "Music"
	GenreMusical     Genre = "Musical"
	GenreMystery     Genre = "Mystery"
	GenreNews        Genre = "News"
	GenreRealityTV   Genre = "Reality-TV"
	GenreRomance     Genre = "Romance"
	GenreSciFi       Genre = "Sci-Fi"
	GenreSport       Genre = "Sport"
	GenreTalkShow    Genre = "Talk-show"
	GenreThriller    Genre = "Thriller"
	GenreWar         Genre = "War"
	GenreWestern     Genre = "Western"
)

type Quality string

const (
	QualityAll       Quality = "All"
	Quality480p      Quality = "480p"
	Quality720p      Quality = "720p"
	Quality1080p     Quality = "1080p"
	Quality1080pX265 Quality = "1080p.x265"
	Quality2160p     Quality = "2160p"
	Quality3D        Quality = "3D"
)

type SortBy string

const (
	SortByTitle         SortBy = "title"
	SortByYear          SortBy = "year"
	SortByRating        SortBy = "rating"
	SortByPeers         SortBy = "peers"
	SortBySeeds         SortBy = "seeds"
	SortByDownloadCount SortBy = "download_count"
	SortByLikeCount     SortBy = "like_count"
	SortByDateAdded     SortBy = "date_added"
)

type OrderBy string

const (
	OrderByAsc  OrderBy = "asc"
	OrderByDesc OrderBy = "desc"
)

var (
	ErrValidationFailure       = errors.New("validation_failure")
	ErrStructValidationFailure = errors.New("struct_validation_failure")
)

type SearchMoviesFilters struct {
	Limit         int     `json:"limit"`
	Page          int     `json:"page"`
	Quality       Quality `json:"quality"`
	MinimumRating int     `json:"minimum_rating"`
	QueryTerm     string  `json:"query_term"`
	Genre         Genre   `json:"genre"`
	SortBy        SortBy  `json:"sort_by"`
	OrderBy       OrderBy `json:"order_by"`
	WithRTRatings bool    `json:"with_rt_ratings"`
}

func DefaultSearchMoviesFilter(query string) *SearchMoviesFilters {
	const (
		defaultPageLimit     = 20
		defaultMinimumRating = 0
	)

	return &SearchMoviesFilters{
		Limit:         defaultPageLimit,
		Page:          1,
		Quality:       QualityAll,
		MinimumRating: 0,
		QueryTerm:     query,
		SortBy:        SortByDateAdded,
		OrderBy:       OrderByDesc,
		WithRTRatings: false,
	}
}

func (f *SearchMoviesFilters) validateFilters() error {
	const (
		maxMinRating = 9
		maxLimit     = 50
	)

	err := validation.ValidateStruct(
		f,
		validation.Field(
			&f.Limit,
			validation.Min(0),
			validation.Max(maxLimit),
		),
		validation.Field(
			&f.Page,
			validation.Min(1),
		),
		validation.Field(
			&f.Quality,
			validation.In(
				QualityAll,
				Quality480p,
				Quality720p,
				Quality1080p,
				Quality1080pX265,
				Quality2160p,
				Quality3D,
			),
		),
		validation.Field(
			&f.MinimumRating,
			validation.Min(0),
			validation.Max(maxMinRating),
		),
		validation.Field(
			&f.Genre,
			validation.In(
				GenreAll,
				GenreAction,
				GenreAdventure,
				GenreAnimation,
				GenreBiography,
				GenreComedy,
				GenreCrime,
				GenreDocumentary,
				GenreDrama,
				GenreFamily,
				GenreFantasy,
				GenreFilmNoir,
				GenreGameShow,
				GenreHistory,
				GenreHorror,
				GenreMusic,
				GenreMusical,
				GenreMystery,
				GenreNews,
				GenreRealityTV,
				GenreRomance,
				GenreSciFi,
				GenreSport,
				GenreTalkShow,
				GenreThriller,
				GenreWar,
				GenreWestern,
			),
		),
		validation.Field(
			&f.SortBy,
			validation.In(
				SortByTitle,
				SortByYear,
				SortByRating,
				SortByPeers,
				SortBySeeds,
				SortByDownloadCount,
				SortByLikeCount,
				SortByDateAdded,
			),
		),
		validation.Field(
			&f.OrderBy,
			validation.In(
				OrderByAsc,
				OrderByDesc,
			),
		),
		validation.Field(
			&f.WithRTRatings,
			validation.In(true, false),
		),
	)

	if err == nil {
		return nil
	}

	return wrapErr(ErrStructValidationFailure, err)
}

func (f *SearchMoviesFilters) getQueryString() (string, error) {
	if err := f.validateFilters(); err != nil {
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
		case Quality, Genre, SortBy, OrderBy:
			str := fmt.Sprintf("%v", v)
			if str != "" {
				queryValues.Add(query, str)
			}
		}
	}

	return queryValues.Encode(), nil
}

type MovieDetailsFilters struct {
	WithImages bool `json:"with_images"`
	WithCast   bool `json:"with_cast"`
}

func DefaultMovieDetailsFilters() *MovieDetailsFilters {
	return &MovieDetailsFilters{
		WithImages: true,
		WithCast:   true,
	}
}

func (f *MovieDetailsFilters) validateFilters() error {
	err := validation.ValidateStruct(
		f,
		validation.Field(
			&f.WithImages,
			validation.In(true, false),
		),
		validation.Field(
			&f.WithCast,
			validation.In(true, false),
		),
	)

	if err == nil {
		return nil
	}

	return wrapErr(ErrStructValidationFailure, err)
}

func (f *MovieDetailsFilters) getQueryString() (string, error) {
	if err := f.validateFilters(); err != nil {
		return "", err
	}

	queryValues := url.Values{}
	if f.WithImages {
		queryValues.Add("with_images", "true")
	}
	if f.WithCast {
		queryValues.Add("with_cast", "true")
	}

	return queryValues.Encode(), nil
}
