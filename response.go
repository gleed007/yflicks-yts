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

type SearchMoviesResponse struct {
	BaseResponse
	Data SearchMoviesData
}

type MovieDetailsResponse struct {
	BaseResponse
	Data MovieDetailsData
}
