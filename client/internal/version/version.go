package version

// Version is stamped at build time:
//
//	go build -ldflags "-X statusphere-client/internal/version.Version=v0.3.0"
var Version = "dev"

func Current() string { return Version }

func IsDev() bool { return Version == "dev" || Version == "" }
