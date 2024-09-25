package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/actions-go/toolkit/core"
	"github.com/mikolajmikolajczyk/semver-sugar/pkg/semver"
	"github.com/mikolajmikolajczyk/semver-sugar/pkg/utils"
)

type ReleaseStrategy string

const (
	ReleaseStrategyRelease = "release"
	ReleaseStrategyTag     = "tag"
	ReleaseStrategyNone    = "none"
)

var (
	ErrEmptyOption                      = errors.New("empty option")
	ErrPRNotClosed                      = errors.New("pull request is not closed")
	ErrPRNotMerged                      = errors.New("pull request is not merged")
	ErrPRNotBase                        = errors.New("missing base ref")
	ErrBaseRefDoesNotMatchReleaseBranch = errors.New("base ref does not match release branch")
	ErrNoValidSemVerLabelFound          = errors.New("no valid semver label found")
)

// ExecuteGuard guards the execution of the action based on the pull request
// state and labels.
func ExecuteGuard(releaseBranch string, eventPath string) error {
	if releaseBranch == "" || eventPath == "" {
		core.Errorf("empty releaseBranch or eventPath: releaseBranch=%s eventPath=%s", releaseBranch, eventPath)
		return ErrEmptyOption // fail
	}

	event := utils.ParseGithubEvent(eventPath)

	if event.Action == nil || *event.Action != "closed" {
		return ErrPRNotClosed // skip
	}
	if event.PullRequest.Merged == nil || !*event.PullRequest.Merged {
		return ErrPRNotMerged // skip
	}

	if event.PullRequest.Base == nil || event.PullRequest.Base.Ref == nil {
		return ErrPRNotBase // here it should fail
	}

	if *event.PullRequest.Base.Ref != releaseBranch {
		return ErrBaseRefDoesNotMatchReleaseBranch // skip

	}
	_, err := semver.ExtractSemVerIncrementFromPullRequest(event.PullRequest)
	if err != nil {
		return ErrNoValidSemVerLabelFound // skip
	}
	return nil
}

func ExecuteGetIncrement(eventPath string) (string, error) {
	event := utils.ParseGithubEvent(eventPath)
	increment, err := semver.ExtractSemVerIncrementFromPullRequest(event.PullRequest)
	return string(increment), err
}

func ExecuteGetGithubLatestTag(repository, githubToken, githubAPIURL, githubUploadsURL, versionRange string) (string, error) {
	return utils.GetGithubLatestTag(repository, githubToken, githubAPIURL, githubUploadsURL, versionRange)
}

func ExecuteGetNextTag(currentVersion, increment, format string) (string, error) {
	version, err := semver.ParseVersion(currentVersion)
	if err != nil {
		return "", err
	}
	inc, err := semver.ParseIncrement(increment)
	if err != nil {
		return "", err
	}
	return version.Bump(inc).Format(format), nil
}

type repository struct {
	owner string
	name  string
	token string
}

func ExecuteCreateRelease(githubRepository, githubSHA, nextTag, githubToken, githubApiUrl, githubUploadsUrl, releaseStrategy string) error {
	parts := strings.Split(githubRepository, "/")
	repo := repository{
		owner: parts[0],
		name:  parts[1],
		token: githubToken,
	}

	ctx := context.Background()
	client, err := utils.NewGithubClient(ctx, githubToken, githubApiUrl, githubUploadsUrl)
	if err != nil {
		return err
	}

	switch releaseStrategy {
	case ReleaseStrategyNone:
		return nil
	case ReleaseStrategyRelease:
		if err := utils.CreateGithubRelease(ctx, client, repo.owner, repo.name, repo.token, nextTag, githubSHA); err != nil {
			return err
		}
	case ReleaseStrategyTag:
		if err := utils.CreateGithubTag(ctx, client, repo.owner, repo.name, repo.token, nextTag, githubSHA); err != nil {
			return err
		}
	default:
		return errors.New("invalid release strategy")
	}

	return nil
}
