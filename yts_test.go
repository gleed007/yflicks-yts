package yts_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	yts "github.com/atifcppprogrammer/yflicks-yts"
)

func TestDefaultTorrentTrackers(t *testing.T) {
	got := yts.DefaultTorrentTrackers()
	want := []string{
		"udp://open.demonii.com:1337/announce",
		"udp://tracker.openbittorrent.com:80",
		"udp://tracker.coppersurfer.tk:6969",
		"udp://glotorrents.pw:6969/announce",
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://torrent.gresille.org:80/announce",
		"udp://p4p.arenabg.com:1337",
		"udp://tracker.leechers-paradise.org:6969",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.DefaultTorrentTrackers() = %v, want %v", got, want)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	got := yts.DefaultClientConfig()
	want := yts.ClientConfig{
		APIBaseURL:      yts.DefaultAPIBaseURL,
		SiteURL:         yts.DefaultSiteURL,
		SiteDomain:      yts.DefaultSiteDomain,
		RequestTimeout:  time.Minute,
		TorrentTrackers: yts.DefaultTorrentTrackers(),
		Debug:           false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.DefaultClientConfig() = %v, want %v", got, want)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	type args struct {
		config *yts.ClientConfig
	}
	tests := []struct {
		name      string
		args      args
		wantErr   error
		wantPanic bool
	}{
		{
			name:      fmt.Sprintf(`panic() if config request timeout < %d`, yts.TimeoutLimitLower),
			args:      args{&yts.ClientConfig{RequestTimeout: time.Second}},
			wantErr:   yts.ErrInvalidClientConfig,
			wantPanic: true,
		},
		{
			name:      fmt.Sprintf(`panic() if config request timeout > %d`, yts.TimeoutLimitUpper),
			args:      args{&yts.ClientConfig{RequestTimeout: time.Hour}},
			wantErr:   yts.ErrInvalidClientConfig,
			wantPanic: true,
		},
		{
			name:      "no panic() if valid client config provided",
			args:      args{&yts.ClientConfig{RequestTimeout: time.Minute}},
			wantErr:   nil,
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				if !tt.wantPanic && recovered == nil {
					return
				}
				if !tt.wantPanic && recovered != nil {
					t.Errorf("yts.NewClientWithConfig() unexpected panic with value %v", recovered)
					return
				}
				if err, _ := recovered.(error); !errors.Is(err, tt.wantErr) {
					t.Errorf("yts.NewClientWithConfig() unexpected panic with error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}()
			yts.NewClientWithConfig(tt.args.config)
		})
	}
}

func TestNewClient(t *testing.T) {
	got := yts.NewClient()
	defaultConfig := yts.DefaultClientConfig()
	want := yts.NewClientWithConfig(&defaultConfig)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("yts.NewClient() = %v, want %v", got, want)
	}
}
