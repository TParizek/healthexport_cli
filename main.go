package main

import "github.com/TParizek/healthexport_cli/cmd"

// Set by goreleaser linker flags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
