package probe

import (
	"fmt"
	"net/http"
)

type HTTPHealthChecker struct {
	Client *http.Client
	URL    string
}

func NewHTTPHealthChecker(addr string) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		Client: &http.Client{Timeout: DefaultTimeout},
		URL:    addr,
	}
}

func (h *HTTPHealthChecker) Check() error {
	resp, err := h.Client.Get(h.URL)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("response is nil")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s (%d)", resp.Status, resp.StatusCode)
	}
	return nil
}
