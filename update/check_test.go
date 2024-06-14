package update

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fynelabs/selfupdate"
	"github.com/stretchr/testify/assert"
)

// func Test_fetchLatestRelease(t *testing.T) {
// 	client := &http.Client{Timeout: 5 * time.Second}
// 	_, err := fetchLatestRelease(client, "https://api.github.com", "dimspell", "gladiator")
// 	assert.NoError(t, err)
// }

func TestGitHubSource_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := []byte("Hello World!")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer ts.Close()

	source := &GitHubSource{
		HttpClient:       &http.Client{Timeout: 5 * time.Second},
		GitHubApiURL:     ts.URL,
		OrganizationName: "test",
		RepositoryName:   "multi",

		exe: GitHubReleaseAsset{
			Name:               "test",
			BrowserDownloadURL: ts.URL,
			ContentType:        "text/plain",
			Label:              "",
		},
	}

	stream, length, err := source.Get(&selfupdate.Version{
		Build:  1,
		Date:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Number: "v1.0.0",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(12), length)

	body, err := io.ReadAll(stream)
	assert.NoError(t, err)
	assert.NoError(t, stream.Close())
	assert.Equal(t, "Hello World!", string(body))
}
