// Package ntpserver version information.
package ntpserver

const (
	// Version is the semantic version of the library.
	Version = "0.2.0"

	VersionMajor = 0
	VersionMinor = 2
	VersionPatch = 0
)

func VersionInfo() string {
	return "go-ntpserver v" + Version
}
