package main

import (
	"os/exec"
	"strings"
)

// execute commandLine
func parseCommand(commandLine string) {
	args := strings.Fields(commandLine)
	cmd := exec.Command(args[0])
	cmd.Args = args
	cmd.Run()
}
