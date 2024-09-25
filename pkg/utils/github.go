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

func NewGithubClient(ctx context.Context, token, githubApiUrl, githubUploadUrl string) (*github.Client, error) {
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

func ParseGithubEvent(filePath string) *github.PullRequestEvent {
	parsed, err := github.ParseWebHook(filePath, readGithubEvent(filePath))
	if err != nil {
		core.Error(err.Error())
	}

	event, ok := parsed.(*github.PullRequestEvent)
	if !ok {
		core.Error("Not a pull request event")
	}
	return event

}

func GetGithubLatestTag(repository, githubToken, githubApiURL, githubUploadsURL, versionRange string) (string, error) {
	ctx := context.Background()

	client, err := NewGithubClient(ctx, githubToken, githubApiURL, githubUploadsURL)
	if err != nil {
		return "", err
	}

	parts := strings.Split(repository, "/")
	owner := parts[0]
	repo := parts[1]

	refs, response, err := client.Git.ListMatchingRefs(ctx, owner, repo, &github.ReferenceListOptions{
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

func CreateGithubTag(ctx context.Context, client *github.Client, owner, repo, token, version, target string) error {
	_, _, err := client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/tags/%s", version)),
		Object: &github.GitObject{
			SHA: &target,
		},
	})
	return err
}

func CreateGithubRelease(ctx context.Context, client *github.Client, owner, repo, token, version, target string) error {
	_, _, err := client.Repositories.CreateRelease(ctx, owner, repo, &github.RepositoryRelease{
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
