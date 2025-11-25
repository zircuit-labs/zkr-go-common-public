# replaceattrmore

A slog handler wrapper that enables 1-to-many attribute transformations, allowing you to expand single log attributes into multiple attributes.

## Installation

```go
import "github.com/zircuit-labs/zkr-go-common/replaceattrmore"
```

## Usage

```go
import (
    "fmt"
    "log/slog"
    "os"
    
    "github.com/zircuit-labs/zkr-go-common/replaceattrmore"
)

// Create a replace function that expands integers into multiple attributes
replaceFunc := func(groups []string, a slog.Attr) []slog.Attr {
    if a.Value.Kind() == slog.KindInt64 {
        val := a.Value.Int64()
        parity := "even"
        if val%2 != 0 {
            parity = "odd"
        }
        return []slog.Attr{
            a, // Keep original int attribute
            slog.String(a.Key+"_hex", fmt.Sprintf("0x%x", val)),
            slog.String(a.Key+"_parity", parity),
        }
    }
    return []slog.Attr{a}
}

// Chain with any slog handler
jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
handler := replaceattrmore.New(jsonHandler, replaceFunc)
logger := slog.New(handler)

// Log with integer attributes
logger.Info("number analysis",
    slog.Int("value", 42),
    slog.Int("count", 7))
```

Output:

```json
{
    "time":"2024-01-01T00:00:00Z",
    "level":"INFO",
    "msg":"number analysis",
    "value":42,
    "value_hex":"0x2a",
    "value_parity":"even",
    "count":7,
    "count_hex":"0x7",
    "count_parity":"odd"
}
```

## Features

- Works with any `slog.Handler` implementation
- Supports `WithGroup` and `WithAttrs` methods
- Allows expanding single attributes into multiple attributes
- Preserves original handler behavior when no transformation is needed
