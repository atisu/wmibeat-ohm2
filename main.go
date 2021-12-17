package main

import (
	"os"

	"github.com/atisu/wmibeat-ohm2/cmd"

	_ "github.com/atisu/wmibeat-ohm2/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
