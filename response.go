package yts

type BaseResponse struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Meta          `json:"@meta"`
}

type SearchMoviesData struct {
	MovieCount int     `json:"movie_count"`
	Limit      int     `json:"limit"`
	PageNumber int     `json:"page_number"`
	Movies     []Movie `json:"movies"`
}

type MovieDetailsData struct {
	Movie MovieDetails `json:"movie"`
}

type MovieSuggestionsData struct {
	MovieCount int     `json:"movie_count"`
	Movies     []Movie `json:"movies"`
}

type SearchMoviesResponse struct {
	BaseResponse
	Data SearchMoviesData
}

type MovieDetailsResponse struct {
	BaseResponse
	Data MovieDetailsData
}

type MovieSuggestionsResponse struct {
	BaseResponse
	Data MovieSuggestionsData
}
