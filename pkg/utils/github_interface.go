package utils

import "github.com/google/go-github/v65/github"

//go:generate mockgen -source=github_interface.go -destination=github_mock.go -package=utils
type GithubActionIface interface {
	CreateGithubTag(version, target string) error
	CreateGithubRelease(version, target string) error
	GetGithubLatestTag(versionRange string) (string, error)
	ParseGithubEvent(filePath string) (*github.PullRequestEvent, error)
	GetIncrementType(eventPath string) (string, error)
	GetNextTag(currentVersion, increment, format string) (string, error)
}
