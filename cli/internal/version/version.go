package version

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
)

func String() string {
	if Version == "dev" {
		return "hangar dev"
	}
	if Commit != "" && Commit != "none" {
		return fmt.Sprintf("hangar %s (%s)", Version, Commit)
	}
	return fmt.Sprintf("hangar %s", Version)
}

func Module() string {
	return "github.com/hangar-sh/hangar"
}
