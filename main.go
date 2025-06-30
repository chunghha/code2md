package main

import (
	"fmt"
	"os"

	cli "code2md/cmd/code2md"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
