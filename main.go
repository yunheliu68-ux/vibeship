package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "collect" {
		runCollect()
		return
	}
	runTUI()
}

func runCollect() {
	fmt.Fprintln(os.Stderr, "vibeship collect: not yet implemented")
	os.Exit(0)
}

func runTUI() {
	fmt.Fprintln(os.Stderr, "vibeship tui: not yet implemented")
	os.Exit(0)
}
