package main

import (
	"net/http"
	"testing"
)

type testCase struct {
	method string
	url    string
	server string
	expected_url []string
}

var reqClient = &Requester{
	&http.Client{},
}
var tests = []testCase{
	{
		"GET",
		"/com",
		"http://localhost:8080",
		[]string{"https://google.com","https://facebook.com"},
	},
	{
		"GET",
		"/ru",
		"http://localhost:8080",
		[]string{"https://mail.ru","https://lenta.ru"},
	},
}
// Test Status Code
func TestResponseCodeRequester(t *testing.T) {
	for _, item := range tests {
		resp, err := reqClient.TestStatusRequester(item.method, item.url, item.server)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			t.Fail()
		}
	}
}

// Test upstream
func TestRequestUrlRequester(t *testing.T) {
	var found = false
	for _, item := range tests {
		resp, err := reqClient.TestStatusRequester(item.method, item.url, item.server)
		if err != nil {
			t.Fatal(err)
		}
		for _, expected := range item.expected_url {
			if resp.Header.Get("Host") == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fail()
		}
	}}