// Package ntpserver version information.
package ntpserver

const (
	// Version is the semantic version of the library.
	Version = "0.3.1"

	VersionMajor = 0
	VersionMinor = 3
	VersionPatch = 1
)

func VersionInfo() string {
	return "go-ntpserver v" + Version
}
