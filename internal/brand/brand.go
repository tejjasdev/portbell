// Package brand holds the project name and the names derived from it, so the
// name lives in one place. Renaming the project means changing this file, the
// module path in go.mod, the cmd/ folder, and the docs.
package brand

const (
	// Name is the command and project name.
	Name = "portbell"
	// Repo is the Go module path, which is also the GitHub address.
	Repo = "github.com/tejjasdev/portbell"
	// URL is the project home page.
	URL = "https://github.com/tejjasdev/portbell"
	// EnvPrefix is the prefix of environment variables a reply may read.
	EnvPrefix = "PORTBELL_"
	// ConfigFile is the config file looked up in the working directory.
	ConfigFile = Name + ".toml"
	// SaveDir is the folder where saved requests are written.
	SaveDir = Name + "-saved"
)
