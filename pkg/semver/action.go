package semver

import (
	"errors"

	"github.com/google/go-github/v65/github"
)

func BumpSemverVersion(version string, increment string, format string) (string, error) {

	v, err := ParseVersion(version)
	if err != nil {
		return "", err
	}
	inc, err := ParseIncrement(increment)
	if err != nil {
		return "", err
	}
	return v.Bump(inc).Format(format), nil
}

func ExtractSemVerIncrementFromPullRequest(pr *github.PullRequest) (Increment, error) {
	validLabelFound := false
	increment := IncrementPatch
	for _, label := range pr.Labels {
		if label.Name == nil {
			continue
		}
		inc, err := ParseIncrement(*label.Name)
		if err != nil {
			continue
		}
		if validLabelFound {
			return increment, errors.New("multiple valid semver labels found")
		}
		validLabelFound = true
		increment = inc
	}
	if !validLabelFound {
		return increment, errors.New("no valid semver labels found")
	}
	return increment, nil
}
