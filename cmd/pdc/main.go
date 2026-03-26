package main

import (
	"os"

	"github.com/matcra587/pagerduty-client/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
