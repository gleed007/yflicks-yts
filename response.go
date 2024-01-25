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
	Data SearchMoviesData `json:"data"`
}

type TrendingMoviesData struct {
	Movies []ScrapedMovie `json:"movies"`
}

type HomePageContentData struct {
	Popular  []ScrapedMovie
	Latest   []ScrapedMovie
	Upcoming []ScrapedUpcomingMovie
}

type MovieDetailsResponse struct {
	BaseResponse
	Data MovieDetailsData `json:"data"`
}

type MovieSuggestionsResponse struct {
	BaseResponse
	Data MovieSuggestionsData `json:"data"`
}

type TrendingMoviesResponse struct {
	Data TrendingMoviesData `json:"data"`
}

type HomePageContentResponse struct {
	Data HomePageContentData `json:"data"`
}
