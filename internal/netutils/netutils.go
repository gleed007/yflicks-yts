package netutils

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

func GetPayload(targetURL string, payload interface{}) error {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	response, err := http.Get(parsedURL.String())
	if err != nil {
		return err
	}

	defer response.Body.Close()
	rawPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(rawPayload, payload)
	if err != nil {
		return err
	}

	return nil
}
