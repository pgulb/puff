package puff

// Repo holds info about repo that a binary can be installed from
type Repo struct {
	Path   string
	Desc   string
	Regexp string
}

// Metadata holds info about a single installed binary and its version
type Metadata struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// MetadataList is used to store all installed bins' metadata on disk
type MetadataList struct {
	Metadata []Metadata `json:"metadata"`
}

// returns fixed list of pre-added repositories
func AvailableRepos() *[]Repo {
	return &[]Repo{
		{
			Path:   "pgulb/plasma",
			Desc:   "Docker container controller with own HTTP API",
			Regexp: `\blinux-amd64\b`,
		},
	}
}
