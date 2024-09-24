package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/actions-go/toolkit/core"
)

func runMain() {
	sleep := os.Getenv("INPUT_MILLISECONDS")
	core.Debug(fmt.Sprintf("Waiting %s milliseconds", sleep))
	core.Debug(time.Now().String())
	delay, err := strconv.Atoi(sleep)
	if err != nil {
		core.Error(err.Error())
		return
	}
	time.Sleep(time.Duration(delay) * time.Millisecond)
	core.Debug(time.Now().String())
	core.SetOutput("time", time.Now().String())
}

func main() {
	runMain()
}
