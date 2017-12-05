package version

import (
	"fmt"
	"runtime"
)

var (
	// SemVer set at build time: -ldflags "-X version.SemVer=1.0.0"
	SemVer string
	// BuildTime set at build time
	BuildTime string
	// GitCommit set at build time
	GitCommit string
)

// ClientVersion contains information about the current client
type ClientVersion struct {
	SemVer    string
	BuildTime string
	GitCommit string
	GoVersion string
	Os        string
	Arch      string
}

// Version constructed at build time
var (
	Version = ClientVersion{
		SemVer,
		BuildTime,
		GitCommit,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	}

	// HumanVersion is a human readable app version
	HumanVersion = fmt.Sprintf("%s\n\tgit-commit: %s\n\tbuild-date: %s\n\tplatform: %s %s %s", Version.SemVer, Version.GitCommit, Version.BuildTime, Version.Os, Version.Arch, Version.GoVersion)

	// ASCIILogo CLI logo
	ASCIILogo = `
    ╦ ╦┌─┐┬─┐┌┬┐┌─┐┌─┐  ╔═╗┌─┐┌┬┐┌─┐┌─┐┬─┐┌─┐┌─┐┬ ┬  ╔╦╗┬─┐┬┌─┐┌─┐┌─┐┬─┐┌─┐
    ╠═╣├┤ ├┬┘│││├┤ └─┐  ║  │ │ ││├┤ ├┤ ├┬┘├┤ └─┐├─┤   ║ ├┬┘││ ┬│ ┬├┤ ├┬┘└─┐
    ╩ ╩└─┘┴└─┴ ┴└─┘└─┘  ╚═╝└─┘─┴┘└─┘└  ┴└─└─┘└─┘┴ ┴   ╩ ┴└─┴└─┘└─┘└─┘┴└─└─┘
    `
)
