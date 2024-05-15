package main

import (
	"encoding/json"
	"fmt"
	"log"
)

func logMethodResponse(name string, response any) {
	payload, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	message := formatMethodReturns(name, string(payload), nil)
	log.Println(message)
}

func formatMethodReturns(name string, response any, err error) string {
	return fmt.Sprintf("client.%s() = %v, %v", name, response, err)
}
