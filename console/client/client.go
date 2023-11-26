package client

import (
	"net/http"
	"time"
)

type Client struct {
	ConsoleAddr string
	HttpClient  *http.Client
}

func New(consoleAddr string) *Client {
	return &Client{
		ConsoleAddr: consoleAddr,
		HttpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}
