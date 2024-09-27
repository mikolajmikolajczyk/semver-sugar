package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	semver "github.com/blang/semver/v4"
	msemver "github.com/mikolajmikolajczyk/semver-sugar/pkg/semver"

	"github.com/google/go-github/v65/github"
	"golang.org/x/oauth2"
)

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
	eventBytes, err := readGithubEvent(filePath)
	if err != nil {
		return nil, err
	}

	parsed, err := github.ParseWebHook("pull_request", eventBytes)
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
	repo, owner, err := parseRepository(impl.Repository)
	if err != nil {
		return "", err
	}
	refs, response, err := impl.GithubClient.Git.ListMatchingRefs(context.Background(), owner, repo, &github.ReferenceListOptions{
		Ref: "tags",
	})
	if err != nil {
		return "", err
	}
	if response != nil && response.StatusCode == http.StatusNotFound {
		return "", errors.New("wrong response when listing mathing refs")
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

func (impl *GithubActionImpl) GetNextTag(currentVersion, increment, format string) (string, error) {
	return msemver.BumpSemverVersion(currentVersion, increment, format)
}

func (impl *GithubActionImpl) CreateGithubTag(version, target string) error {
	repo, owner, err := parseRepository(impl.Repository)
	if err != nil {
		return err
	}

	_, _, err = impl.GithubClient.Git.CreateRef(context.Background(), owner, repo, &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/tags/%s", version)),
		Object: &github.GitObject{
			SHA: &target,
		},
	})
	return err
}

func (impl *GithubActionImpl) CreateGithubRelease(version, target string) error {
	repo, owner, err := parseRepository(impl.Repository)
	if err != nil {
		return err
	}
	_, _, err = impl.GithubClient.Repositories.CreateRelease(context.Background(), owner, repo, &github.RepositoryRelease{
		Name:            &version,
		TagName:         &version,
		TargetCommitish: &target,
		Draft:           github.Bool(false),
		Prerelease:      github.Bool(false),
	})
	return err
}

func (impl *GithubActionImpl) GetIncrementType(eventPath string) (string, error) {
	event, err := impl.ParseGithubEvent(eventPath)
	if err != nil {
		return "", err
	}
	increment, err := msemver.ExtractSemVerIncrementFromPullRequest(event.PullRequest)
	return string(increment), err
}

func readGithubEvent(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return b, nil
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

func parseRepository(repository string) (owner string, repo string, err error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s, expected 'owner/repo'", repository)
	}
	return parts[0], parts[1], nil
}
