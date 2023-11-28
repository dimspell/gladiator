package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func unmarshalResponse[T any](body []byte, statusCode int, err error) (T, error) {
	var t T
	if err != nil {
		return t, err
	}
	if statusCode > 399 {
		return t, fmt.Errorf("status code equal to %d", statusCode)
	}
	if err := json.Unmarshal(body, &t); err != nil {
		return t, err
	}
	return t, nil
}

func doRequest(
	ctx context.Context,
	client *http.Client,
	method string,
	url string,
	headers map[string]string,
	body any,
) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		if method == "POST" || method == "PUT" {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, 0, err
			}

			bodyReader = bytes.NewBuffer(bodyBytes)
		}
	}

	// Create the new request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, err
	}

	// Set request headers
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return payload, resp.StatusCode, nil
}
