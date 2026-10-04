package main

import "strings"

func greet(name string) string {
	nameWithoutSpace := strings.TrimSpace(name)
	if nameWithoutSpace == "" {
		nameWithoutSpace = "World"
	}
	return "Hello, " + nameWithoutSpace + "!"
}
