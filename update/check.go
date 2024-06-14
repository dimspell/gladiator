package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fynelabs/selfupdate"
)

var _ selfupdate.Source = (*GitHubSource)(nil)

type GitHubSource struct {
	HttpClient       *http.Client
	GitHubApiURL     string
	OrganizationName string
	RepositoryName   string

	checksum GitHubReleaseAsset
	exe      GitHubReleaseAsset
}

func NewGitHubSource(client *http.Client, orgName, repoName string) *GitHubSource {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &GitHubSource{
		HttpClient:       client,
		GitHubApiURL:     "https://api.github.com",
		OrganizationName: orgName,
		RepositoryName:   repoName,
	}
}

func (g *GitHubSource) LatestVersion() (*selfupdate.Version, error) {
	release, err := fetchLatestRelease(g.HttpClient, g.GitHubApiURL, g.OrganizationName, g.RepositoryName)
	if err != nil {
		return nil, fmt.Errorf("could not fetch latest release: %w", err)
	}

	for _, asset := range release.Assets {
		if asset.ContentType == "application/octet-stream" && strings.HasSuffix(asset.Name, ".exe") {
			g.exe = asset
		} else if (asset.ContentType == "application/json") && asset.Name == "checksum.json" {
			g.checksum = asset
		}
	}

	if g.exe.BrowserDownloadURL == "" || g.checksum.BrowserDownloadURL == "" {
		return nil, fmt.Errorf("could not find download URL for update")
	}

	return &selfupdate.Version{
		Build:  1, // TODO: Fetch build number
		Number: strings.TrimPrefix(release.TagName, "v"),
		Date:   release.PublishedAt,
	}, nil
}

func (g *GitHubSource) GetSignature() ([64]byte, error) {
	sign := [64]byte{}
	return sign, fmt.Errorf("not implemented")
}

func (g *GitHubSource) Get(version *selfupdate.Version) (io.ReadCloser, int64, error) {
	if g.exe.BrowserDownloadURL == "" {
		return nil, 0, fmt.Errorf("unknown download URL")
	}

	response, err := g.HttpClient.Get(g.exe.BrowserDownloadURL)
	if err != nil {
		return nil, 0, err
	}
	return response.Body, response.ContentLength, nil
}

type GitHubRelease struct {
	TagName     string               `json:"tag_name"`
	PreRelease  bool                 `json:"prerelease"`
	PublishedAt time.Time            `json:"published_at"`
	Assets      []GitHubReleaseAsset `json:"assets"`
}

type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
	Label              string `json:"label"`
}

func fetchLatestRelease(httpClient *http.Client, host, owner, repo string) (GitHubRelease, error) {
	link := fmt.Sprintf("%s/repos/%s/%s/releases/latest", host, owner, repo)
	resp, err := httpClient.Get(link)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("error fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("error reading response body: %w", err)
	}

	var release GitHubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		return GitHubRelease{}, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return release, nil
}
