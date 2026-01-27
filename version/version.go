package version

import (
	"encoding/json"
	"os"
	"time"
)

type VersionInformation struct {
	GitCommit string    `json:"git_commit"`
	GitDate   int64     `json:"git_date"`
	GitDirty  bool      `json:"git_dirty"`
	Version   string    `json:"version"`
	Variant   string    `json:"variant"`
	Date      time.Time `json:"-"`
}

func (v VersionInformation) Commit() string {
	if v.GitCommit == "" || v.GitCommit == "unknown" {
		return "unknown"
	}
	if v.GitDirty {
		return v.GitCommit + "-dirty"
	}
	return v.GitCommit
}

func (v VersionInformation) ShortCommit() string {
	if v.GitCommit == "" || v.GitCommit == "unknown" {
		return "unknown"
	}
	if len(v.GitCommit) > 7 {
		return v.GitCommit[:7]
	}
	return v.GitCommit
}

func (v VersionInformation) LogValues() []any {
	return []any{
		"git_commit", v.Commit(),
		"git_date", v.Date,
		"version", v.Version,
		"variant", v.Variant,
	}
}

func (v VersionInformation) VersionCommit() string {
	if v.Version == "" || v.Version == "unknown" {
		return "unknown"
	}
	vc := v.Version
	if c := v.ShortCommit(); c != "" && c != "unknown" {
		vc = vc + "-" + c
	}
	if r := v.Variant; r != "" {
		vc = vc + "-" + r
	}
	return vc
}

var Info VersionInformation

func init() {
	// Read the version information from the JSON file
	file, err := os.ReadFile("/etc/version.json")
	if err != nil {
		return
	}
	err = json.Unmarshal(file, &Info)
	if err != nil {
		return
	}
	Info.Date = time.Unix(Info.GitDate, 0).UTC()
}
