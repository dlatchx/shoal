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

// compare two []byte
func testEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func insertRule(slice []*RoutingRule, index int, value *RoutingRule) []*RoutingRule {
	newSlice := make([]*RoutingRule, len(slice), cap(slice)+1)
	copy(newSlice, slice)
	slice = newSlice[0 : len(slice)+1]
	copy(slice[index+1:], slice[index:])
	slice[index] = value
	return slice
}
