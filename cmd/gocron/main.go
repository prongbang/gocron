package main

import (
	"github.com/prongbang/gocron/cmd/gocron/runner"
	_ "time/tzdata"
)

func main() {
	runner.Start()
}
