# stores

The `stores` package provides abstractions and implementations for managing storage interactions with various backend systems. Current implementations include S3-compatible storage and PostgreSQL utilities.

## Overview

This package provides:

- **S3 BlobStore** - Interface for S3-compatible object storage (AWS S3, MinIO, etc.)
- **PostgreSQL utilities** - Cursor-based pagination and database helpers
- Configuration-driven setup with support for multiple environments
- Error handling with rich context information

## Sub-packages

### s3

S3-compatible blob storage implementation.

### pg

PostgreSQL database utilities, particularly for pagination.

## S3 BlobStore

### Configuration

```go
type BlobStoreConfig struct {
    Endpoint        string `koanf:"endpoint"`         // S3 endpoint URL (for MinIO/custom)
    AccessKeyID     string `koanf:"accesskeyid"`      // Access key ID
    SecretAccessKey string `koanf:"secretaccesskey"`  // Secret access key
    Bucket          string `koanf:"bucket"`           // S3 bucket name
    Region          string `koanf:"region"`           // AWS region

    // MinIO-specific settings
    S3ForcePathStyle bool `koanf:"s3forcepathstyle"` // true for MinIO, false for AWS
    DisableSSL       bool `koanf:"disablessl"`       // true for MinIO, false for AWS
}
```

### Basic Usage

```go
import (
    "context"
    "github.com/zircuit-labs/zkr-go-common/stores/s3"
)

// Create from configuration
config := s3.BlobStoreConfig{
    Region:          "us-west-2",
    Bucket:          "my-app-storage",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
}

store, err := s3.NewBlobStoreFromConfig(config)
if err != nil {
    return fmt.Errorf("failed to create blob store: %w", err)
}

// Store data
ctx := context.Background()
data := []byte("Hello, World!")
err = store.Put(ctx, "files/hello.txt", data)
if err != nil {
    return fmt.Errorf("failed to store data: %w", err)
}

// Retrieve data
retrievedData, err := store.Get(ctx, "files/hello.txt")
if err != nil {
    return fmt.Errorf("failed to retrieve data: %w", err)
}

// Check if exists
exists, err := store.Exists(ctx, "files/hello.txt")
if err != nil {
    return errcontext.Add(
        stacktrace.Wrap(err),
        slog.String("operation", "check_existence"),
        slog.String("key", "files/hello.txt"),
    )
}

// Delete data
err = store.Delete(ctx, "files/hello.txt")
if err != nil {
    return errcontext.Add(
        stacktrace.Wrap(err),
        slog.String("operation", "delete_data"),
        slog.String("key", "files/hello.txt"),
    )
}
```

### Configuration Integration

```toml
# config/storage.toml
[blobstore]
endpoint = ""                    # Empty for AWS, set for MinIO
region = "us-west-2"
bucket = "my-app-storage"
accesskeyid = "${AWS_ACCESS_KEY_ID}"
secretaccesskey = "${AWS_SECRET_ACCESS_KEY}"
s3forcepathstyle = false        # true for MinIO
disablessl = false              # true for MinIO over HTTP
```

```go
import (
    "github.com/zircuit-labs/zkr-go-common/config"
    "github.com/zircuit-labs/zkr-go-common/stores/s3"
)

func setupStorage(cfg *config.Configuration) (*s3.BlobStore, error) {
    var storeConfig s3.BlobStoreConfig
    if err := cfg.Unmarshal("blobstore", &storeConfig); err != nil {
        return nil, err
    }

    return s3.NewBlobStoreFromConfig(storeConfig)
}
```

### MinIO Configuration

For local development or MinIO deployments:

```toml
[blobstore]
endpoint = "http://localhost:9000"
region = "us-east-1"              # Required but can be any value for MinIO
bucket = "test-bucket"
accesskeyid = "minioadmin"
secretaccesskey = "minioadmin"
s3forcepathstyle = true           # Required for MinIO
disablessl = true                 # For HTTP endpoints
```

### Error Handling

The S3 store provides specific error types:

```go
import "github.com/zircuit-labs/zkr-go-common/stores/s3"

_, err := store.Get(ctx, "nonexistent-file")
if errors.Is(err, s3.ErrNotFound) {
    // Handle not found case
    log.Println("File does not exist")
} else if err != nil {
    // Handle other errors
    log.Printf("Storage error: %v", err)
}
```

## PostgreSQL Utilities

### Cursor-based Pagination

The `pg` package provides efficient cursor-based pagination for PostgreSQL queries.

```go
import (
    "github.com/zircuit-labs/zkr-go-common/stores/pg"
    "github.com/uptrace/bun"
)

// Define your model
type User struct {
    ID        int64     `bun:"id,pk,autoincrement"`
    Email     string    `bun:"email"`
    CreatedAt time.Time `bun:"created_at"`
}

// Paginate users
func listUsers(db *bun.DB, cursor pg.Cursor, limit int) ([]User, pg.Cursor, error) {
    var users []User

    query := db.NewSelect().Model(&users).Order("created_at DESC", "id DESC")

    // Apply cursor pagination
    pagedQuery, err := pg.ApplyCursor(query, cursor, []string{"created_at", "id"})
    if err != nil {
        return nil, pg.Cursor{}, err
    }

    // Execute query with limit
    err = pagedQuery.Limit(limit + 1).Scan(ctx) // +1 to check for more pages
    if err != nil {
        return nil, pg.Cursor{}, err
    }

    // Generate next cursor
    nextCursor := pg.Cursor{}
    if len(users) > limit {
        users = users[:limit] // Remove the extra record
        nextCursor.Next = pg.EncodeCursor(users[limit-1].CreatedAt, users[limit-1].ID)
    }

    return users, nextCursor, nil
}
```

### Cursor Types

```go
type Cursor struct {
    Next     string  // Token for next page
    Previous string  // Token for previous page
}

// Check if cursor indicates reverse pagination
if cursor.IsReverse() {
    // Handle previous page
}

// Check if cursor exists
if cursor.Exists() {
    // Apply cursor to query
}
```

## Integration Examples

### With Runner and Config

```go
//go:embed config
var configFS embed.FS

func runService(cfg *config.Configuration, tm runner.Runner, logger *slog.Logger) error {
    // Setup S3 storage
    var s3Config s3.BlobStoreConfig
    if err := cfg.Unmarshal("blobstore", &s3Config); err != nil {
        return errcontext.Add(stacktrace.Wrap(err), slog.String("config_section", "blobstore"))
    }

    store, err := s3.NewBlobStoreFromConfig(s3Config)
    if err != nil {
        return stacktrace.Wrap(err)
    }

    // Setup database
    var dbConfig DatabaseConfig
    if err := cfg.Unmarshal("database", &dbConfig); err != nil {
        return errcontext.Add(stacktrace.Wrap(err), slog.String("config_section", "database"))
    }

    db := setupDatabase(dbConfig)
    tm.Cleanup(func() { db.Close() })

    // Use in your service
    service := &MyService{
        store:  store,
        db:     db,
        logger: logger,
    }

    // Register with HTTP handler, etc.
    return nil
}
```

### With HTTP Handlers

```go
type FileHandler struct {
    store  *s3.BlobStore
    logger *slog.Logger
}

func (h *FileHandler) UploadFile(c echo.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.JSON(400, map[string]string{"error": "No file uploaded"})
    }

    src, err := file.Open()
    if err != nil {
        return c.JSON(500, map[string]string{"error": "Failed to open file"})
    }
    defer src.Close()

    data, err := io.ReadAll(src)
    if err != nil {
        return c.JSON(500, map[string]string{"error": "Failed to read file"})
    }

    key := fmt.Sprintf("uploads/%s", file.Filename)
    err = h.store.Put(c.Request().Context(), key, data)
    if err != nil {
        h.logger.Error("Failed to store file", slog.Any("error", err))
        return c.JSON(500, map[string]string{"error": "Storage failed"})
    }

    return c.JSON(200, map[string]string{
        "message": "File uploaded successfully",
        "key":     key,
    })
}

func (h *FileHandler) ListUsers(c echo.Context) error {
    // Parse cursor from query parameters
    cursor := pg.Cursor{
        Next:     c.QueryParam("next"),
        Previous: c.QueryParam("previous"),
    }

    limit := 20 // Default page size

    users, nextCursor, err := h.listUsersWithPagination(c.Request().Context(), cursor, limit)
    if err != nil {
        h.logger.Error("Failed to list users", slog.Any("error", err))
        return c.JSON(500, map[string]string{"error": "Database query failed"})
    }

    return c.JSON(200, map[string]interface{}{
        "users":  users,
        "cursor": nextCursor,
    })
}
```

## Environment Variables

The stores package works well with environment variable substitution:

```bash
# AWS credentials
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=us-west-2

# Database connection
export DATABASE_URL=postgres://user:password@localhost/dbname

# Storage bucket
export STORAGE_BUCKET=my-app-storage
```

## Best Practices

1. **Use configuration management** - Leverage the config package for all storage settings
2. **Handle errors appropriately** - Check for specific error types (e.g., `s3.ErrNotFound`)
3. **Use context for cancellation** - All storage operations should accept context
4. **Implement proper cleanup** - Register cleanup functions with the task manager
5. **Use cursor pagination** - For efficient pagination of large datasets
6. **Log storage operations** - Include relevant context in error logs
7. **Configure for environment** - Use different settings for development vs production

## Dependencies

- `github.com/aws/aws-sdk-go-v2` - AWS S3 client
- `github.com/uptrace/bun` - PostgreSQL ORM
- Zircuit's `config` package for configuration management
- Zircuit's `xerrors` packages for error handling
