package version

var Version = "dev"

func Current() string { return Version }

func IsDev() bool { return Version == "dev" || Version == "" }
