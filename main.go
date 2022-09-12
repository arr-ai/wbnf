package main

import "github.com/arr-ai/wbnf/cmd"

// Version   - Binary version
// GitCommit - Commit SHA of the source code
// BuildDate - Binary build date
// BuildOS   - Operating System used to build binary
//
//nolint:gochecknoglobals
var (
	Version   = "unspecified"
	GitCommit = "unspecified"
	BuildDate = "unspecified"
	BuildOS   = "unspecified"
)

func main() {
	cmd.Main(cmd.VersionTags{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		BuildOS:   BuildOS,
	})
}
