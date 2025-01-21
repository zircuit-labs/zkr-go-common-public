# ZKR-GO-COMMON

This repo is a collection of packages that are useful to Zircuit beyond the confines of zkr-monorepo. At some future point in time, some or all of these packages may be open-sourced, which may or may not happen as a collective or individually.

Each sub-package _should_ contain its own README with details and examples of usage.

## Sub-Packages

| Package    | Description |
| -----------|-------------|
| calm       | Recover from a panic as an error with a stacktrace. |
| config     | Parse configuration information from files and environment variables. |
| http       | Serve HTTP as a task. |
| log        | Zircuit's Go logger. Uses slog.Logger to write though Zerolog and output meaningful JSON logs. Contains special parsing for errors wrapped through xerrors package. |
| messagebus | Interact with NATS in a streamlined way. Create consumers as tasks. |
| retry      | Highly customizable retry functionality. |
| stores     | Manage storage interactions. Current implementations: S3. |
| task       | Easily manage multiple goroutines in the form of tasks. |
| version    | Parse version information from a local file. |
| xerrors    | Wrap errors with additional type-safe data using generics. Sub-packages for stacktraces, adding loggable context, and defined error classifications. |
