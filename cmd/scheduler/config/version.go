package config

import "fmt"

const (
	appName    = "indexer-scheduler"
	appVersion = "0.1.0"
	gitCommit  = "-"
	goVersion  = "-"
)

// VersionString returns the full app version string
func VersionString() string {
	return fmt.Sprintf(
		"%s %s (git: %s, %s)",
		appName,
		appVersion,
		gitCommit,
		goVersion,
	)
}
