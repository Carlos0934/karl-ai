package main

import (
	"fmt"
	"os"

	"github.com/carlos0934/karl-ai/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.New(version, os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
