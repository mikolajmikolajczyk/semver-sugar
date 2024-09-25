package main

import (
	"fmt"
	"os"

	"github.com/actions-go/toolkit/core"
)

func main() {
	release_branch := os.Getenv("INPUT_RELEASE_BRANCH")
	if release_branch == "" {
		core.SetFailed("INPUT_RELEASE_BRANCH is not set")
		return
	}
	core.Info(fmt.Sprintf("Release branch is %s", release_branch))
}
