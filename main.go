package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/actions-go/toolkit/core"
	"github.com/mikolajmikolajczyk/semver-sugar/pkg/utils"
)

var osExit = os.Exit

func Exit(code int) {
	core.Info(fmt.Sprintf("Exiting with code: %v", code))
	osExit(code)
}

type ActionConfig struct {
	ReleaseBranch    string
	ReleaseStrategy  string
	NextTag          string
	TagFormat        string
	GithubApiUrl     string
	GithubUploadsUrl string
	CustomReleaseSHA string
	VersionRange     string
	Increment        string
	EventPath        string
	GithubRepository string
	GithubToken      string
	CurrentTag       string
}

func ActionConfigFromEnv() ActionConfig {
	githubSHA := os.Getenv("GITHUB_SHA")
	customReleaseSHA := os.Getenv("INPUT_CUSTOM_RELEASE_SHA")
	if customReleaseSHA != "" {
		githubSHA = customReleaseSHA
	}
	return ActionConfig{
		ReleaseBranch:    os.Getenv("INPUT_RELEASE_BRANCH"),
		ReleaseStrategy:  os.Getenv("INPUT_RELEASE_STRATEGY"),
		NextTag:          os.Getenv("INPUT_NEXT_TAG"),
		TagFormat:        os.Getenv("INPUT_TAG_FORMAT"),
		GithubApiUrl:     os.Getenv("INPUT_GITHUB_API_URL"),
		GithubUploadsUrl: os.Getenv("INPUT_GITHUB_UPLOADS_URL"),
		CustomReleaseSHA: githubSHA,
		VersionRange:     os.Getenv("INPUT_VERSION_RANGE"),
		Increment:        os.Getenv("INPUT_INCREMENT"),
		EventPath:        os.Getenv("GITHUB_EVENT_PATH"),
		GithubRepository: os.Getenv("GITHUB_REPOSITORY"),
		GithubToken:      os.Getenv("GITHUB_TOKEN"),
		CurrentTag:       "",
	}
}

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
func executeGuard(ghActionIface utils.GithubActionIface, releaseBranch string, eventPath string) error {
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
	_, err = ghActionIface.GetIncrementType(eventPath)
	if err != nil {
		return ErrNoValidSemVerLabelFound // here it should fail
	}
	return nil
}

func executeCreateRelease(ghActionIface utils.GithubActionIface, githubSHA, nextTag, releaseStrategy string) error {
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

func executeAction(ghActionIface utils.GithubActionIface, actionConfig ActionConfig) {
	core.Info("Executing PR guard now")
	// This will prevent the action from running if the guard fails
	err := executeGuard(ghActionIface, actionConfig.ReleaseBranch, actionConfig.EventPath)
	if err != nil {
		core.Error(err.Error())
		if err == ErrPRNotBase || err == ErrEmptyOption || err == ErrNoValidSemVerLabelFound {
			Exit(1)
		}
		Exit(0)
	}

	core.Debug("Executing next tag calculation now")
	if actionConfig.NextTag == "" {
		core.Debug("Getting latest tag from github repository")
		latestTag, err := ghActionIface.GetGithubLatestTag(actionConfig.VersionRange)
		if err != nil {
			core.Error(err.Error())
			Exit(1)
		}
		core.Debug("Latest tag is: " + latestTag)
		actionConfig.CurrentTag = latestTag
		core.Debug("Getting increment type from github event")
		incr, err := ghActionIface.GetIncrementType(actionConfig.EventPath)
		if err != nil {
			core.Error(err.Error())
			Exit(1)
		}
		core.Debug("Increment type is: " + string(incr))
		actionConfig.Increment = string(incr)
		core.Debug("Getting next tag from latest tag and increment type")
		nextTag, err := ghActionIface.GetNextTag(latestTag, actionConfig.Increment, actionConfig.TagFormat)
		if err != nil {
			core.Error(err.Error())
			Exit(1)
		}
		core.Debug("Next tag is: " + nextTag)
		actionConfig.NextTag = nextTag
	}

	core.Debug("Executing release creation now")
	err = executeCreateRelease(ghActionIface, actionConfig.CustomReleaseSHA, actionConfig.NextTag, actionConfig.ReleaseStrategy)
	if err != nil {
		core.Error(err.Error())
		Exit(1)
	}
	core.SetOutput("tag", actionConfig.NextTag)
	core.SetOutput("increment", actionConfig.Increment)
	core.Infof("Release strategy was: %v, tag was: %v and next tag created was: %v, increment was: %v\n", actionConfig.ReleaseStrategy, actionConfig.CurrentTag, actionConfig.NextTag, actionConfig.Increment)
}
func main() {
	actionConfig := ActionConfigFromEnv()
	ghIface, err := utils.NewGithubActionImpl(actionConfig.GithubRepository, actionConfig.GithubToken, actionConfig.GithubApiUrl, actionConfig.GithubUploadsUrl)
	if err != nil {
		core.Error(err.Error())
		os.Exit(1)
	}

	executeAction(ghIface, actionConfig)
}
