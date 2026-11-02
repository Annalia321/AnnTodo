package version

// Version is overridden at build time with -ldflags "-X anntodo/pkg/version.Version=x.y.z".
var Version = "0.1.0"

// Commit is the git commit the binary was built from (optional).
var Commit = ""
