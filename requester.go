package main

import (
	"net/http"
)

type Requester struct {
	client *http.Client
}

func (reqClient *Requester) TestStatusRequester(method, url string, server string) (*http.Response, error) {
	req, err := http.NewRequest(method, server + url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := reqClient.client.Do(req)
	return resp, err
}