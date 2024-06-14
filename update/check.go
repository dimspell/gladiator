package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fynelabs/selfupdate"
)

var _ selfupdate.Source = (*GitHubSource)(nil)

type GitHubSource struct {
	LastRelease GitHubRelease
	Err         error
}

func NewGitHubSource(ctx context.Context, orgName, repoName string) *GitHubSource {
	release, err := checkLatestRelease(ctx, orgName, repoName)
	return &GitHubSource{LastRelease: release, Err: err}
}

func (g *GitHubSource) GetSignature() ([64]byte, error) {
	// TODO implement me
	panic("implement me")
}

func (g *GitHubSource) LatestVersion() (*selfupdate.Version, error) {
	// TODO implement me
	panic("implement me")
}

func (g *GitHubSource) Get(version *selfupdate.Version) (io.ReadCloser, int64, error) {
	// TODO implement me
	panic("implement me")
}

type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	PreRelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`

	Assets []GitHubReleaseAsset `json:"assets"`
}

type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

func checkLatestRelease(ctx context.Context, owner, repo string) (GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("error fetching latest release: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("error fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("error reading response body: %w", err)
	}
	fmt.Println(string(body))

	var release GitHubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return release, nil
}
