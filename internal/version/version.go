package version

var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildDate = "unknown"
)

func GetVersion() string {
    return Version
}

func GetFullVersion() string {
	if GitCommit != "unknown" {
		return Version + " (" + GitCommit + ")"
	}
	return Version
}
