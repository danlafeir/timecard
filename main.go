package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danlafeir/timecard/cmd"
	"github.com/danlafeir/timecard/internal/selfupdate"
)

// Build metadata set at build time via -ldflags. See Makefile.
var (
	BuildVersion = "dev"
	BuildGitHash = "dev"
	BuildDate    = "unknown"
)

func checkUpgrade() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	checkFile := filepath.Join(home, ".timecard", "upgrade-check")
	os.MkdirAll(filepath.Dir(checkFile), 0o755)

	today := time.Now().Format("2006-01-02")
	var lastDate string
	if f, err := os.Open(checkFile); err == nil {
		fmt.Fscanf(f, "%s", &lastDate)
		f.Close()
	}
	if lastDate == today {
		return
	}

	release, err := selfupdate.LatestRelease()
	if err == nil && release != nil && selfupdate.IsNewer(release.TagName, BuildVersion) {
		fmt.Fprintf(os.Stderr, "A new version of timecard is available (%s). Run 'timecard update' to upgrade.\n", release.TagName)
	}

	f, err := os.Create(checkFile)
	if err == nil {
		fmt.Fprintf(f, "%s", today)
		f.Close()
	}
}

func main() {
	checkUpgrade()
	cmd.BuildVersion = BuildVersion
	cmd.BuildGitHash = BuildGitHash
	cmd.BuildDate = BuildDate
	cmd.Execute()
}
