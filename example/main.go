package main

import (
	"errors"
	"log"

	yts "github.com/atifcppprogrammer/yflicks-yts"
)

var client *yts.Client

func init() {
	config := yts.DefaultClientConfig()
	config.Debug = true
	client, _ = yts.NewClientWithConfig(&config)
}

func main() {
	var methodCallers = []func() error{
		trendingMovies,
		homePageContent,
		MovieSuggestions,
		MovieDetails,
		SearchMovies,
	}

	for _, caller := range methodCallers {
		if err := caller(); err != nil {
			log.Fatal(err)
		}
	}
}

func homePageContent() error {
	const methodName = "HomePageContent"
	response, err := client.HomePageContent()
	if err != nil {
		message := formatMethodReturns(methodName, response, err)
		return errors.New(message)
	}

	logMethodResponse(methodName, response)
	return nil
}

func trendingMovies() error {
	const methodName = "TrendingMovies"
	response, err := client.TrendingMovies()
	if err != nil {
		message := formatMethodReturns(methodName, response, err)
		return errors.New(message)
	}

	logMethodResponse(methodName, response)
	return nil
}

func MovieSuggestions() error {
	const methodName = "MovieSuggestions"
	const movieID = 3175
	response, err := client.MovieSuggestions(movieID)
	if err != nil {
		message := formatMethodReturns(methodName, response, err)
		return errors.New(message)
	}

	logMethodResponse(methodName, response)
	return nil
}

func MovieDetails() error {
	const methodName = "MovieDetails"
	const movieID = 3175
	filters := yts.DefaultMovieDetailsFilters()
	response, err := client.MovieDetails(movieID, filters)
	if err != nil {
		message := formatMethodReturns(methodName, response, err)
		return errors.New(message)
	}

	logMethodResponse(methodName, response)
	return nil
}

func SearchMovies() error {
	const methodName = "SearchMovies"
	filters := yts.DefaultSearchMoviesFilters("oppenheimer")
	response, err := client.SearchMovies(filters)
	if err != nil {
		message := formatMethodReturns(methodName, response, err)
		return errors.New(message)
	}

	logMethodResponse(methodName, response)
	return nil
}
