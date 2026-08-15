package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/carlos0934/karl-ai/cli"
)

var version = "dev"

func main() {
	if err := cli.New(version, os.Stdout, os.Stderr).Execute(); err != nil {
		if !errors.Is(err, cli.ErrValidationFailed) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
