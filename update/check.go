package update

import (
	"github.com/fynelabs/selfupdate"
	"io"
)

var _ selfupdate.Source = (*GitHubSource)(nil)

type GitHubSource struct {
	OrganizationName string
	RepositoryName   string
}

func NewGitHubSource(orgName, repoName string) *GitHubSource {
	return &GitHubSource{
		OrganizationName: orgName,
		RepositoryName:   repoName,
	}
}

func (g *GitHubSource) GetSignature() ([64]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GitHubSource) LatestVersion() (*selfupdate.Version, error) {
	//TODO implement me
	panic("implement me")
}

func (g *GitHubSource) Get(version *selfupdate.Version) (io.ReadCloser, int64, error) {
	//TODO implement me
	panic("implement me")
}
