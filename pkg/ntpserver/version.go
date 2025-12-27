// Package ntpserver version information.
package ntpserver

const (
	// Version is the semantic version of the library.
	Version = "0.1.0"

	VersionMajor = 0
	VersionMinor = 1
	VersionPatch = 0
)

func VersionInfo() string {
	return "go-ntpserver v" + Version
}
