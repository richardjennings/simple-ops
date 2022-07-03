package main

import (
	"github.com/richardjennings/simple-ops/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
