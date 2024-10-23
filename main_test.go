package main

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	github "github.com/google/go-github/v65/github"
	"github.com/mikolajmikolajczyk/semver-sugar/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestExecuteCreateRelease(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGHActionIface := utils.NewMockGithubActionIface(ctrl)

	tests := []struct {
		name            string
		releaseStrategy string
		setupMock       func()
		expectedError   error
	}{
		{
			name:            "Release strategy None",
			releaseStrategy: ReleaseStrategyNone,
			setupMock:       func() {}, // No expectations since no calls should be made
			expectedError:   nil,
		},
		{
			name:            "Successful Release strategy Release",
			releaseStrategy: ReleaseStrategyRelease,
			setupMock: func() {
				// Expect a successful call to CreateGithubRelease
				mockGHActionIface.EXPECT().CreateGithubRelease("v1.0.0", "abc123").Return(nil)
			},
			expectedError: nil,
		},
		{
			name:            "Failed Release strategy Release",
			releaseStrategy: ReleaseStrategyRelease,
			setupMock: func() {
				// Expect CreateGithubRelease to return an error
				mockGHActionIface.EXPECT().CreateGithubRelease("v1.0.0", "abc123").Return(errors.New("release creation failed"))
			},
			expectedError: errors.New("release creation failed"),
		},
		{
			name:            "Successful Release strategy Tag",
			releaseStrategy: ReleaseStrategyTag,
			setupMock: func() {
				// Expect a successful call to CreateGithubTag
				mockGHActionIface.EXPECT().CreateGithubTag("v1.0.0", "abc123").Return(nil)
			},
			expectedError: nil,
		},
		{
			name:            "Failed Release strategy Tag",
			releaseStrategy: ReleaseStrategyTag,
			setupMock: func() {
				// Expect CreateGithubTag to return an error
				mockGHActionIface.EXPECT().CreateGithubTag("v1.0.0", "abc123").Return(errors.New("tag creation failed"))
			},
			expectedError: errors.New("tag creation failed"),
		},
		{
			name:            "Invalid Release strategy",
			releaseStrategy: "invalid",
			setupMock:       func() {}, // No expectations for invalid strategy
			expectedError:   errors.New("invalid release strategy"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			err := executeCreateRelease(mockGHActionIface, "abc123", "v1.0.0", tt.releaseStrategy)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func TestExecuteGuard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGHActionIface := utils.NewMockGithubActionIface(ctrl)

	tests := []struct {
		name          string
		releaseBranch string
		eventPath     string
		setupMock     func()
		expectedError error
	}{
		{
			name:          "empty releaseBranch or eventPath",
			releaseBranch: "",
			eventPath:     "test_event.json",
			setupMock:     func() {},
			expectedError: ErrEmptyOption,
		},
		{
			name:          "Error parsing GitHub event",
			releaseBranch: "main",
			eventPath:     "test_event.json",
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(nil, errors.New("parsing error"))
			},
			expectedError: errors.New("parsing error"),
		},
		{
			name:          "PR not closed",
			releaseBranch: "main",
			eventPath:     "test_event.json",
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action: github.String("open"),
				}, nil)
			},
			expectedError: ErrPRNotClosed,
		},
		{
			name:          "PR not merged",
			releaseBranch: "main",
			eventPath:     "test_event.json",
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(false)},
				}, nil)
			},
			expectedError: ErrPRNotMerged,
		},
		{
			name:          "PR base ref missing",
			releaseBranch: "main",
			eventPath:     "test_event.json",
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: nil},
				}, nil)
			},
			expectedError: ErrPRNotBase,
		},
		{
			name:          "Base ref mismatch",
			releaseBranch: "main",
			eventPath:     "test_event.json",
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("develop")}},
				}, nil)
			},
			expectedError: ErrBaseRefDoesNotMatchReleaseBranch,
		},
		{
			name:          "No valid SemVer label found",
			releaseBranch: "main",
			eventPath:     "test_event.json",
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().GetIncrementType(gomock.Any()).Return("", ErrNoValidSemVerLabelFound)
			},
			expectedError: ErrNoValidSemVerLabelFound,
		},
		{
			name:          "Successful execution with valid SemVer label",
			releaseBranch: "main",
			eventPath:     "test_event.json",
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().GetIncrementType(gomock.Any()).Return("patch", nil)
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			err := executeGuard(mockGHActionIface, tt.releaseBranch, tt.eventPath)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func TestExecuteAction(t *testing.T) {
	osExit = func(code int) {
		panic(code)
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGHActionIface := utils.NewMockGithubActionIface(ctrl)

	// Variable to capture exit code via panic
	var exitCode int
	defer func() {
		if r := recover(); r != nil {
			if code, ok := r.(int); ok {
				exitCode = code
			} else {
				t.Fatalf("expected exit code to be of type int, but got: %v", r)
			}
		}
	}()

	tests := []struct {
		name          string
		actionConfig  ActionConfig
		setupMock     func()
		expectedExit  int
		expectedError string
	}{
		{
			name: "Successful execution without NextTag",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("patch", nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("patch", nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().GetNextTag(gomock.Any(), gomock.Any(), gomock.Any()).Return("v1.0.1", nil)
				mockGHActionIface.EXPECT().GetGithubLatestTag(gomock.Any()).Return("v1.0.0", nil)
				mockGHActionIface.EXPECT().CreateGithubRelease("v1.0.1", "abc123").Return(nil)
			},
			expectedExit: 0,
		},
		{
			name: "Successful execution without NextTag - skip-release enabled",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("patch", nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("patch", nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(true, nil)
				mockGHActionIface.EXPECT().GetNextTag(gomock.Any(), gomock.Any(), gomock.Any()).Return("v1.0.1", nil)
				mockGHActionIface.EXPECT().GetGithubLatestTag(gomock.Any()).Return("v1.0.0", nil)
				mockGHActionIface.EXPECT().CreateGithubRelease(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedExit: 0,
		},
		{
			name: "Guard fails with ErrEmptyOption",
			actionConfig: ActionConfig{
				ReleaseBranch:    "",
				EventPath:        "test_event.json",
				NextTag:          "v1.0.0",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
			},
			expectedExit:  1,
			expectedError: ErrEmptyOption.Error(),
		},
		{
			name: "Guard fails with ErrPRNotBase",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "v1.0.0",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(nil, ErrPRNotBase)
			},
			expectedExit:  1,
			expectedError: ErrPRNotBase.Error(),
		},
		{
			name: "Guard skip error",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "v1.0.0",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(nil, ErrPRNotClosed)
			},
			expectedExit: 0,
		},
		{
			name: "Successful execution with existing NextTag",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "v1.0.1",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("patch", nil)
				mockGHActionIface.EXPECT().CreateGithubRelease("v1.0.1", "abc123").Return(nil)

			},
			expectedExit: 0,
		},
		{
			name: "Successful execution with existing NextTag - skip-release enabled",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "v1.0.1",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(true, nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("patch", nil)
				mockGHActionIface.EXPECT().CreateGithubRelease(gomock.Any(), gomock.Any()).Times(0)

			},
			expectedExit: 0,
		},
		{
			name: "Error fetching latest tag",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "",
				VersionRange:     ">=1.0.0",
				TagFormat:        "v%d.%d.%d",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("patch", nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().GetGithubLatestTag(">=1.0.0").Return("", errors.New("failed to get latest tag"))
			},
			expectedExit:  1,
			expectedError: "failed to get latest tag",
		},
		{
			name: "Error getting increment",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "",
				VersionRange:     ">=1.0.0",
				TagFormat:        "v%d.%d.%d",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action: github.String("closed"),
					PullRequest: &github.PullRequest{Labels: []*github.Label{
						{
							Name: github.String("qwerty"),
						},
					}, Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().GetGithubLatestTag(">=1.0.0").Return("v1.0.0", nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("", errors.New("failed to get increment"))
			},
			expectedExit:  1,
			expectedError: "failed to get increment",
		},
		{
			name: "Error generating next tag",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "",
				VersionRange:     ">=1.0.0",
				TagFormat:        "v%d.%d.%d",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("minor", nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("minor", nil)
				mockGHActionIface.EXPECT().GetNextTag("v1.0.0", "minor", "v%d.%d.%d").Return("", errors.New("failed to generate next tag"))
			},
			expectedExit:  1,
			expectedError: "failed to generate next tag",
		},
		{
			name: "Error creating the release",
			actionConfig: ActionConfig{
				ReleaseBranch:    "main",
				EventPath:        "test_event.json",
				NextTag:          "v1.1.0",
				ReleaseStrategy:  ReleaseStrategyRelease,
				CustomReleaseSHA: "abc123",
			},
			setupMock: func() {
				mockGHActionIface.EXPECT().ParseGithubEvent("test_event.json").Return(&github.PullRequestEvent{
					Action:      github.String("closed"),
					PullRequest: &github.PullRequest{Merged: github.Bool(true), Base: &github.PullRequestBranch{Ref: github.String("main")}},
				}, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skip-release", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().DoesLabelExist("skipRelease", gomock.Any()).Return(false, nil)
				mockGHActionIface.EXPECT().GetIncrementType("test_event.json").Return("minor", nil)
				mockGHActionIface.EXPECT().CreateGithubRelease("v1.1.0", "abc123").Return(errors.New("failed to create release"))
			},
			expectedExit:  1,
			expectedError: "failed to create release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			// Reset exitCode before each test
			exitCode = 0
			defer func() {
				if r := recover(); r != nil {
					if code, ok := r.(int); ok {
						exitCode = code
					} else {
						t.Fatalf("expected exit code to be of type int, but got: %v", r)
					}
				}
			}()

			// Execute the action
			executeAction(mockGHActionIface, tt.actionConfig)
			// Assert the exit code
			assert.Equal(t, tt.expectedExit, exitCode)
		})
	}
}
