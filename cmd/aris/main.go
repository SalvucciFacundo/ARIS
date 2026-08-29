package main

import (
	"fmt"
	"os"

	"aris/internal/adapters/ui/cli"
)

func main() {
	runner, err := cli.NewRunner()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing ARIS: %v\n", err)
		os.Exit(1)
	}
	defer runner.Close()

	code := runner.Execute(os.Args)
	if code != 0 {
		os.Exit(code)
	}
}
