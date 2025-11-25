# version

The `version` package provides utilities for parsing and accessing version information from a local file, enabling applications to report their build and deployment metadata.

## Overview

This package:

- Reads version information from `/etc/version.json`
- Provides structured access to git commit, version, and build metadata
- Integrates with logging systems for version reporting
- Supports build variant identification
- Handles dirty git working tree detection

## Core Type

### VersionInformation

```go
type VersionInformation struct {
    GitCommit string    `json:"git_commit"` // Git commit hash
    GitDate   int64     `json:"git_date"`   // Git commit date (Unix timestamp)
    GitDirty  bool      `json:"git_dirty"`  // Whether working tree was dirty
    Version   string    `json:"version"`    // Application version
    Variant   string    `json:"variant"`    // Build variant (e.g., "debug", "release")
    Date      time.Time `json:"-"`          // Parsed git date as time.Time
}
```

## Usage

### Accessing Version Information

```go
import (
    "fmt"
    "github.com/zircuit-labs/zkr-go-common/version"
)

func main() {
    // Access global version information
    info := version.Info

    fmt.Printf("Version: %s\n", info.Version)
    fmt.Printf("Git Commit: %s\n", info.Commit())
    fmt.Printf("Build Date: %s\n", info.Date.Format(time.RFC3339))
    fmt.Printf("Variant: %s\n", info.Variant)
}
```

## File Location

The package expects the version file at `/etc/version.json`. This location:

- Is standard for system-level configuration
- Works well in Docker containers
- Is accessible by applications running as different users
- Can be easily mounted or copied during deployment

If you need to customize the location, you would need to modify the package or create a wrapper that loads from a different path.

## Dependencies

This package has no external dependencies and only uses Go standard library packages:

- `encoding/json` - For parsing the version JSON file
- `os` - For file system access
- `time` - For date/time handling
