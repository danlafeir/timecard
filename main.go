package main

import "github.com/danlafeir/timecard/cmd"

// BuildGitHash is set at build time via -ldflags
var BuildGitHash = "dev"

// BuildLatestHash is set at build time via -ldflags to the latest available hash
var BuildLatestHash = "dev"

func main() {
	cmd.Execute()
}
