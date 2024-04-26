package yts

// A Meta represents the meta information provided as part of the response of the
// following YTS API endpoints.
//
// - "/api/v2/list_movies.json"
// - "/api/v2/movie_details.json"
// - "/api/v2/movie_suggestions.json"
type Meta struct {
	ServerTime     int    `json:"server_time"`
	ServerTimezone string `json:"server_timezone"`
	APIVersion     int    `json:"api_version"`
	ExecutionTime  string `json:"execution_time"`
}

// A Cast represents the cast information provided as part of the response of the
// following YTS API endpoints.
//
// - "/api/v2/list_movies.json"
// - "/api/v2/movie_details.json"
// - "/api/v2/movie_suggestions.json"
type Cast struct {
	Name          string `json:"name"`
	CharacterName string `json:"character_name"`
	ImdbCode      string `json:"imdb_code"`
	URLSmallImage string `json:"url_small_image"`
}

// A Torrent represents the torrent information provided as part of the response of
// the following YTS API endpoints.
//
// - "/api/v2/list_movies.json"
// - "/api/v2/movie_details.json"
// - "/api/v2/movie_suggestions.json"
type Torrent struct {
	URL              string  `json:"url"`
	Hash             string  `json:"hash"`
	Quality          Quality `json:"quality"`
	Type             string  `json:"type"`
	IsRepack         string  `json:"is_repack"`
	VideoCodec       string  `json:"video_codec"`
	BitDepth         string  `json:"bit_depth"`
	AudioChannels    string  `json:"audio_channels"`
	Seeds            int     `json:"seeds"`
	Peers            int     `json:"peers"`
	Size             string  `json:"size"`
	SizeBytes        int     `json:"size_bytes"`
	DateUploaded     string  `json:"date_uploaded"`
	DateUploadedUnix int     `json:"date_uploaded_unix"`
}

// A TorrentInfo serves no grander purpose than a *TorrentInfo being the return type
// of the GetTorrentInfo method of the TorrrentInfoGetter interface.
type TorrentInfo struct {
	MovieTitle string
	Torrents   []Torrent
}

// A TorrentInfoGetter serves essentially serves as a union type and is implemented
// by the MoviePartial type so that instances of both Movie and MovieDetails can be
// provided as inputs to the MagnetLinks method of a yts.Client
type TorrentInfoGetter interface {
	GetTorrentInfo() *TorrentInfo
}

// A MoviePartial represents the common movie information provided as part of the
// response of the following YTS API endpoints.
//
// - "/api/v2/list_movies.json"
// - "/api/v2/movie_details.json"
// - "/api/v2/movie_suggestions.json"
type MoviePartial struct {
	ID                      int       `json:"id"`
	URL                     string    `json:"url"`
	ImdbCode                string    `json:"imdb_code"`
	Title                   string    `json:"title"`
	TitleEnglish            string    `json:"title_english"`
	TitleLong               string    `json:"title_long"`
	Slug                    string    `json:"slug"`
	Year                    int       `json:"year"`
	Rating                  float64   `json:"rating"`
	Runtime                 int       `json:"runtime"`
	Genres                  []Genre   `json:"genres"`
	DescriptionFull         string    `json:"description_full"`
	YtTrailerCode           string    `json:"yt_trailer_code"`
	Language                string    `json:"language"`
	MpaRating               string    `json:"mpa_rating"`
	BackgroundImage         string    `json:"background_image"`
	BackgroundImageOriginal string    `json:"background_image_original"`
	SmallCoverImage         string    `json:"small_cover_image"`
	MediumCoverImage        string    `json:"medium_cover_image"`
	LargeCoverImage         string    `json:"large_cover_image"`
	Torrents                []Torrent `json:"torrents"`
	DateUploaded            string    `json:"date_uploaded"`
	DateUploadedUnix        int       `json:"date_uploaded_unix"`
}

func (mp *MoviePartial) GetTorrentInfo() *TorrentInfo {
	return &TorrentInfo{mp.TitleLong, mp.Torrents}
}

// A Movie represents the movie information provided as part of the response of the
// following YTS API endpoints.
//
// - "/api/v2/list_movies.json"
// - "/api/v2/movie_suggestions.json"
type Movie struct {
	MoviePartial
	Summary  string `json:"summary"`
	Synopsis string `json:"synopsis"`
	State    string `json:"state"`
}

// A MoviePartial represents the movie information provided as part of the response
// of the following YTS API endpoints.
//
// - "/api/v2/movie_details.json"
type MovieDetails struct {
	MoviePartial
	LikeCount              int    `json:"like_count"`
	DescriptionIntro       string `json:"description_intro"`
	MediumScreenshotImage1 string `json:"medium_screenshot_image1"`
	MediumScreenshotImage2 string `json:"medium_screenshot_image2"`
	MediumScreenshotImage3 string `json:"medium_screenshot_image3"`
	LargeScreenshotImage1  string `json:"large_screenshot_image1"`
	LargeScreenshotImage2  string `json:"large_screenshot_image2"`
	LargeScreenshotImage3  string `json:"large_screenshot_image3"`
	Cast                   []Cast `json:"cast"`
}
