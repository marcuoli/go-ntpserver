// Package ntpserver version information.
package ntpserver

const (
	// Version is the semantic version of the library.
	Version = "0.3.2"

	VersionMajor = 0
	VersionMinor = 3
	VersionPatch = 2
)

func VersionInfo() string {
	return "go-ntpserver v" + Version
}
