package main

import (
	"os"

	"github.com/godatei/datei/internal/cmd"
)

func main() {
	if err := cmd.NewCLI().Execute(); err != nil {
		os.Exit(1)
	}
}
