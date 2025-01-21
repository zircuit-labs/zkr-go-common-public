package version

import (
	"encoding/json"
	"os"
	"time"
)

type Information struct {
	GitCommit    string    `json:"git_commit"`
	GitDate      string    `json:"git_date"`
	GitBranch    string    `json:"git_branch"`
	Version      string    `json:"version"`
	Meta         string    `json:"meta"`
	ProverCommit string    `json:"prover_commit"`
	L2GethCommit string    `json:"l2geth_commit"`
	Date         time.Time `json:"-"`
}

var Info Information

func init() {
	file, err := os.ReadFile("/etc/version.json")
	if err != nil {
		return
	}
	err = json.Unmarshal(file, &Info)
	if err != nil {
		return
	}
	Info.Date = parseDate(Info.GitDate)
}

func parseDate(s string) time.Time {
	d, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return d.UTC()
}
