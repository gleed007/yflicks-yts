/*
Package yts provides a client that exposes methods for interacting with the YTS API
(https://yts.mx/api), as well as scraping the YTS website, for content for which
there are no API endpoints provided. This package was developed keeping the yflicks
desktop application in mind, however you can leverage this package to create your own
projects as well.

The general workflow for this package involves creating a client instance like so:

	client, err := yts.NewClient()

Or if you wish to provide a custom configuration for the resulting client this can
be done in the following manner.

	var (
		parsedSiteURL, _    = url.Parse(DefaultSiteURL)
		parsedAPIBaseURL, _ = url.Parse(DefaultAPIBaseURL)
		torrentTrackers     =  []string{
		  "udp://tracker.openbittorrent.com:80",
		  "udp://open.demonii.com:1337/announce",
		  "udp://tracker.coppersurfer.tk:6969",
		}
	)

	config := yts.ClientConfig{
		APIBaseURL:      *parsedAPIBaseURL,
		SiteURL:         *parsedSiteURL,
		RequestTimeout:  time.Minute,
		TorrentTrackers: torrentTrackers,
		Debug:           false,
	}

	client, err := NewClientWithConfig(&config)

In most situations however it is prudent to use the default configuration and then
modifying it to suit your needs.

	config := DefaultClientConfig()
	config.Debug = true
	config.RequestTimeout = time.Minute * 2
	client, err := NewClientWithConfig(&config)

With the the *yts.Client instance instantiated you can leverage the methods provided
by the client in the following manner.

	filters := yts.DefaultSearchMoviesFilters("oppenheimer")
	response, err := client.SearchMovies(filters)
	...
	filters := yts.DefaultMovieDetailsFilters()
	response, err := client.MovieDetails(3175, filters)
	...
	response, err := client.MovieSuggestions(3175)
	...
	slug := "oppenheimer-2023"
	id, err := client.ResolveMovieSlugToID(slug)
	...
	response, err := client.TrendingMovies()
	...
	response, err := client.HomePageContent()
	...
	slug := "oppenheimer-2023"
	response, err := client.MovieDirector(slug)
	...
	response, err := client.MovieReviews(slug)
	...
	page := 1
	slug := "oppenheimer-2023"
	response, err := client.MovieComments(slug, page)
	...
	slug := "oppenheimer-2023"
	response, err := client.MovieAdditionalDetails(slug)

See the accompanying example program for a more detailed tutorial on how to use this
package.
*/
package yts
