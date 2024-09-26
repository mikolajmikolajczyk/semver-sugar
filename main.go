package main

import (
	"os"

	"github.com/actions-go/toolkit/core"
	"github.com/mikolajmikolajczyk/semver-sugar/pkg/commands"
	"github.com/mikolajmikolajczyk/semver-sugar/pkg/utils"
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
	}
}

func executeAction(ghActionIface utils.GithubActionIface, actionConfig ActionConfig) {
	// This will prevent the action from running if the guard fails
	err := commands.ExecuteGuard(ghActionIface, actionConfig.ReleaseBranch, actionConfig.EventPath)
	if err != nil {
		if err == commands.ErrPRNotBase || err == commands.ErrEmptyOption {
			core.Error(err.Error())
			os.Exit(1)
		}
		core.Info(err.Error())
		os.Exit(0)
	}

	if actionConfig.NextTag != "" {
		latestTag, err := commands.ExecuteGetGithubLatestTag(ghActionIface, actionConfig.VersionRange)
		if err != nil {
			core.Error(err.Error())
			os.Exit(1)
		}
		incr, err := commands.ExecuteGetIncrement(ghActionIface, actionConfig.EventPath)
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

	err = commands.ExecuteCreateRelease(ghActionIface, actionConfig.CustomReleaseSHA, actionConfig.NextTag, actionConfig.ReleaseStrategy)
	if err != nil {
		core.Error(err.Error())
		os.Exit(1)
	}
	core.SetOutput("tag", actionConfig.NextTag)
	core.SetOutput("increment", actionConfig.Increment)
}

func main() {
	actionConfig := ActionConfigFromEnv()
	ghIface, err := utils.NewGithubActionImpl(actionConfig.GithubApiUrl, actionConfig.GithubUploadsUrl, actionConfig.GithubRepository, actionConfig.GithubToken)
	if err != nil {
		core.Error(err.Error())
		os.Exit(1)
	}

	executeAction(ghIface, actionConfig)
}
