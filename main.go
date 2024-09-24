package main

import (
	"fmt"
	"os"

	"github.com/actions-go/toolkit/core"
)

func main() {
	release_branch := os.Getenv("INPUT_RELEASE_BRANCH")
	core.Debug(fmt.Sprintf("Release branch is %s", release_branch))
}
