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
