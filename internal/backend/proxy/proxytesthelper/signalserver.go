package proxytesthelper

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/dimspell/gladiator/console/signalserver"
)

func StartSignalServer(t testing.TB) string {
	t.Helper()

	h, err := signalserver.NewServer()
	if err != nil {
		t.Fatal(err)
		return ""
	}
	ts := httptest.NewServer(h)

	t.Cleanup(func() {
		ts.Close()
	})

	wsURI, _ := url.Parse(ts.URL)
	wsURI.Scheme = "ws"

	return wsURI.String()
}
