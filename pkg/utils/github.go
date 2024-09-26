package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/actions-go/toolkit/core"
	semver "github.com/blang/semver/v4"

	"github.com/google/go-github/v65/github"
	"golang.org/x/oauth2"
)

type GithubActionIface interface {
	CreateGithubTag(version, target string) error
	CreateGithubRelease(version, target string) error
	GetGithubLatestTag(versionRange string) (string, error)
	ParseGithubEvent(filePath string) (*github.PullRequestEvent, error)
}

type GithubActionImpl struct {
	Repository       string
	Token            string
	GithubApiUrl     string
	GithubUploadsUrl string
	GithubClient     *github.Client
}

func NewGithubActionImpl(repository, token, githubApiUrl, githubUploadsUrl string) (*GithubActionImpl, error) {
	ghClient, err := newGithubClient(context.Background(), token, githubApiUrl, githubUploadsUrl)
	return &GithubActionImpl{
		Repository:       repository,
		Token:            token,
		GithubApiUrl:     githubApiUrl,
		GithubUploadsUrl: githubUploadsUrl,
		GithubClient:     ghClient,
	}, err
}

func (impl *GithubActionImpl) ParseGithubEvent(filePath string) (*github.PullRequestEvent, error) {
	parsed, err := github.ParseWebHook(filePath, readGithubEvent(filePath))
	if err != nil {
		return nil, err
	}

	event, ok := parsed.(*github.PullRequestEvent)
	if !ok {
		return nil, fmt.Errorf("invalid event")
	}
	return event, nil

}

func (impl *GithubActionImpl) GetGithubLatestTag(versionRange string) (string, error) {
	ctx := context.Background()

	parts := strings.Split(impl.Repository, "/")
	owner := parts[0]
	repo := parts[1]

	refs, response, err := impl.GithubClient.Git.ListMatchingRefs(ctx, owner, repo, &github.ReferenceListOptions{
		Ref: "tags",
	})
	if response != nil && response.StatusCode == http.StatusNotFound {
		return "v0.0.0", nil
	}
	if err != nil {
		return "", err
	}
	expectedRange, err := semver.ParseRange(versionRange)
	if err != nil {
		return "", err
	}

	latest := semver.MustParse("0.0.0")
	for _, ref := range refs {
		version, err := semver.ParseTolerant(strings.Replace(*ref.Ref, "refs/tags/", "", 1))
		if err != nil {
			continue
		}
		if expectedRange(version) && version.GT(latest) {
			latest = version
		}
	}
	return latest.String(), nil
}

func (impl *GithubActionImpl) CreateGithubTag(version, target string) error {
	parts := strings.Split(impl.Repository, "/")
	owner := parts[0]
	repo := parts[1]

	_, _, err := impl.GithubClient.Git.CreateRef(context.Background(), owner, repo, &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/tags/%s", version)),
		Object: &github.GitObject{
			SHA: &target,
		},
	})
	return err
}

func (impl *GithubActionImpl) CreateGithubRelease(version, target string) error {
	parts := strings.Split(impl.Repository, "/")
	owner := parts[0]
	repo := parts[1]
	_, _, err := impl.GithubClient.Repositories.CreateRelease(context.Background(), owner, repo, &github.RepositoryRelease{
		Name:            &version,
		TagName:         &version,
		TargetCommitish: &target,
		Draft:           github.Bool(false),
		Prerelease:      github.Bool(false),
	})
	return err
}

func readGithubEvent(filePath string) []byte {
	file, err := os.Open(filePath)
	if err != nil {
		core.Error(err.Error())
	}
	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		core.Error(err.Error())
	}
	core.Info(string(b))
	return b
}

func newGithubClient(ctx context.Context, token, githubApiUrl, githubUploadUrl string) (*github.Client, error) {
	var err error
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(ctx, tokenSource))
	if githubApiUrl != "" {
		if githubUploadUrl == "" {
			githubUploadUrl = strings.Replace(githubApiUrl, "api", "uploads", 1)
		}
		client, err = client.WithEnterpriseURLs(githubApiUrl, githubUploadUrl)
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}
