package yts

type Movie struct {
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
	Genres                  []string  `json:"genres"`
	DescriptionFull         string    `json:"description_full"`
	Summary                 string    `json:"summary"`
	Synopsis                string    `json:"synopsis"`
	State                   string    `json:"state"`
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

type Torrent struct {
	URL              string `json:"url"`
	Hash             string `json:"hash"`
	Quality          string `json:"quality"`
	Type             string `json:"type"`
	IsRepack         string `json:"is_repack"`
	VideoCodec       string `json:"video_codec"`
	BitDepth         string `json:"bit_depth"`
	AudioChannels    string `json:"audio_channels"`
	Seeds            int    `json:"seeds"`
	Peers            int    `json:"peers"`
	Size             string `json:"size"`
	SizeBytes        int    `json:"size_bytes"`
	DateUploaded     string `json:"date_uploaded"`
	DateUploadedUnix int    `json:"date_uploaded_unix"`
}

type Meta struct {
	ServerTime     int    `json:"server_time"`
	ServerTimezone string `json:"server_timezone"`
	APIVersion     int    `json:"api_version"`
	ExecutionTime  string `json:"execution_time"`
}
