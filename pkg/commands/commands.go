package commands

import (
	"errors"

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
func ExecuteGuard(ghActionIface utils.GithubActionIface, releaseBranch string, eventPath string) error {
	if releaseBranch == "" || eventPath == "" {
		core.Errorf("empty releaseBranch or eventPath: releaseBranch=%s eventPath=%s", releaseBranch, eventPath)
		return ErrEmptyOption // fail
	}

	event, err := ghActionIface.ParseGithubEvent(eventPath)
	if err != nil {
		return err
	}

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
	_, err = semver.ExtractSemVerIncrementFromPullRequest(event.PullRequest)
	if err != nil {
		return ErrNoValidSemVerLabelFound // skip
	}
	return nil
}

func ExecuteGetIncrement(ghActionIface utils.GithubActionIface, eventPath string) (string, error) {
	event, err := ghActionIface.ParseGithubEvent(eventPath)
	if err != nil {
		return "", err
	}
	increment, err := semver.ExtractSemVerIncrementFromPullRequest(event.PullRequest)
	return string(increment), err
}

func ExecuteGetGithubLatestTag(ghActionIface utils.GithubActionIface, versionRange string) (string, error) {
	return ghActionIface.GetGithubLatestTag(versionRange)
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

func ExecuteCreateRelease(ghActionIface utils.GithubActionIface, githubSHA, nextTag, releaseStrategy string) error {
	switch releaseStrategy {
	case ReleaseStrategyNone:
		return nil
	case ReleaseStrategyRelease:
		if err := ghActionIface.CreateGithubRelease(nextTag, githubSHA); err != nil {
			return err
		}
	case ReleaseStrategyTag:
		if err := ghActionIface.CreateGithubTag(nextTag, githubSHA); err != nil {
			return err
		}
	default:
		return errors.New("invalid release strategy")
	}

	return nil
}
