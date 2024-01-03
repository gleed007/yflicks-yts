package netutils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestEmployee struct {
	Name   string `json:"name"`
	Salary int    `json:"salary"`
}

func getTestHandlerFuncFor(payload interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serialized, _ := json.Marshal(payload)
		fmt.Fprintf(w, "%s", serialized)
	}
}

func TestGetPayload(t *testing.T) {
	t.Run("returns error if ill-formed URL provided as argument", func(t *testing.T) {
		malformedURL := "proto://malformed-url.com"
		received := GetPayload(malformedURL, struct{}{})
		expected := fmt.Errorf(`Get "%s": unsupported protocol scheme "proto"`, malformedURL)
		if received == nil || received.Error() != expected.Error() {
			t.Errorf("received error %s, expected %s", received, expected)
		}
	})

	t.Run("populates passed struct with response payload from server endpoint", func(t *testing.T) {
		expected := TestEmployee{"employee", 5000}
		handler := getTestHandlerFuncFor(expected)
		server := httptest.NewServer(handler)
		defer server.Close()

		received := TestEmployee{}
		err := GetPayload(server.URL, &received)
		if err != nil {
			t.Errorf("received error %s, expected %v", err, nil)
		}

		if received.Name != expected.Name {
			t.Errorf("received name %s, expected %s", received.Name, expected.Name)
		}

		if received.Salary != expected.Salary {
			t.Errorf("received salary %d, expected %d", received.Salary, expected.Salary)
		}
	})
}
