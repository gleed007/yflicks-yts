package yts

import (
	"fmt"
	"net/url"
	"testing"
)

func TestGetMagnet(t *testing.T) {
	t.Run("returns error if no torrent found for given quality", func(t *testing.T) {
		moviePartial := MoviePartial{
			Torrents: []Torrent{
				{Quality: Quality1080p},
				{Quality: Quality2160p},
			},
		}

		quality := Quality3D
		_, received := moviePartial.GetMagnet(quality)
		expected := fmt.Errorf("no torrent found having quality %s", quality)
		if received == nil || received.Error() != expected.Error() {
			t.Errorf("received error %v, expected %v", received, expected)
		}
	})

	t.Run("returns correct magnet link when torrent with given quality is found", func(t *testing.T) {
		torrent := Torrent{
			Hash:    "CDED33F7FBF3E4E073778848FAD17674C0A35B82",
			Quality: Quality1080p,
		}

		moviePartial := MoviePartial{
			TitleLong: "Oppenheimer (2023)",
			Torrents:  []Torrent{torrent},
		}

		quality := Quality1080p
		received, err := moviePartial.GetMagnet(quality)
		defaultTrackers := DefaultTorrentTrackerList()
		trackers := url.Values{}

		for _, tracker := range defaultTrackers {
			trackers.Add("tr", tracker)
		}

		movieName := fmt.Sprintf(
			"%s+[%s]+[YTS.MX]",
			moviePartial.TitleLong,
			moviePartial.Torrents[0].Quality,
		)

		expected := fmt.Sprintf(
			"magnet:?xt=urn:btih:%s&dn=%s&%s",
			moviePartial.Torrents[0].Hash,
			url.QueryEscape(movieName),
			trackers.Encode(),
		)

		if err != nil {
			t.Errorf("received error %v, expected %v", err, nil)
		}

		if received != expected {
			t.Errorf("received magnet %s, expected %s", received, expected)
		}
	})
}
