package main

import (
	"os"

	"github.com/actions-go/toolkit/core"
	"github.com/mikolajmikolajczyk/semver-sugar/pkg/commands"
)

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
	GuardDisabled    bool
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
		GuardDisabled:    os.Getenv("INPUT_GUARD_DISABLED") == "true",
	}
}

func main() {
	actionConfig := ActionConfigFromEnv()

	// This will prevent the action from running if the guard fails
	err := commands.ExecuteGuard(actionConfig.ReleaseBranch, actionConfig.EventPath)
	if err != nil {
		if err == commands.ErrPRNotBase || err == commands.ErrEmptyOption {
			core.Error(err.Error())
			os.Exit(1)
		}
		core.Info(err.Error())
		os.Exit(0)
	}

	if actionConfig.NextTag != "" {
		latestTag, err := commands.ExecuteGetGithubLatestTag(actionConfig.GithubRepository, actionConfig.GithubToken, actionConfig.GithubApiUrl, actionConfig.GithubUploadsUrl, actionConfig.VersionRange)
		if err != nil {
			core.Error(err.Error())
			os.Exit(1)
		}
		incr, err := commands.ExecuteGetIncrement(actionConfig.EventPath)
		if err != nil {
			core.Error(err.Error())
			os.Exit(1)
		}
		actionConfig.Increment = string(incr)
		nextTag, err := commands.ExecuteGetNextTag(latestTag, actionConfig.Increment, actionConfig.TagFormat)
		if err != nil {
			core.Error(err.Error())
			os.Exit(1)
		}
		actionConfig.NextTag = nextTag
	}

	err = commands.ExecuteCreateRelease(actionConfig.GithubRepository, actionConfig.CustomReleaseSHA, actionConfig.NextTag, actionConfig.GithubToken, actionConfig.GithubApiUrl, actionConfig.GithubUploadsUrl, actionConfig.ReleaseStrategy)
	if err != nil {
		core.Error(err.Error())
		os.Exit(1)
	}
	core.SetOutput("tag", actionConfig.NextTag)
	core.SetOutput("increment", actionConfig.Increment)
}
