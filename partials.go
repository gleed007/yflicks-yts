package yts

type Meta struct {
	ServerTime     int    `json:"server_time"`
	ServerTimezone string `json:"server_timezone"`
	APIVersion     int    `json:"api_version"`
	ExecutionTime  string `json:"execution_time"`
}

type Cast struct {
	Name          string `json:"name"`
	CharacterName string `json:"character_name"`
	ImdbCode      string `json:"imdb_code"`
	URLSmallImage string `json:"url_small_image"`
}

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

type TorrentInfo struct {
	MovieTitle string
	Torrents   []Torrent
}

type TorrentInfoGetter interface {
	GetTorrentInfo() *TorrentInfo
}

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

type Movie struct {
	MoviePartial
	Summary  string `json:"summary"`
	Synopsis string `json:"synopsis"`
	State    string `json:"state"`
}

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
