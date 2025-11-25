# zkr-go-common

This repo is a collection of Go packages that are useful to Zircuit in multiple other repositories.
Each sub-package should contain its own README with details and examples of usage.

## Zircuit Bug Bounty Program

This repository is subject to the Zircuit Bug Bounty Program. Please see [SECURITY.md](SECURITY.md) for more information.

## Sub-Packages

| Package    | Description |
| -----------|-------------|
| calm       | Recover from a panic as an error with a stacktrace. |
| collections| Generic data structures (Set) with iterator and functional programming support. |
| config     | Parse configuration information from files and environment variables. |
| http       | Serve HTTP as a task. |
| iter       | Functional operations on Go 1.23+ iterators (Filter, Transform). |
| log        | Zircuit's Go logger. Uses standard library to output meaningful JSON logs. Contains special parsing for errors that implement slog.LogValuer |
| messagebus | Interact with NATS in a streamlined way. Create consumers as tasks. |
| replaceattrmore | A slog handler wrapper that enables 1-to-many attribute transformations. |
| retry      | Highly customizable retry functionality. |
| runner     | Boilerplate abstraction for standardized services. |
| singleton  | Distributed locking backed by NATS KV store, which uses fencing to ensure correctness. The lock has limited time validity, and will extend that validity itself while locked. |
| stores     | Manage storage interactions. Current implementations: S3. |
| task       | Easily manage multiple goroutines in the form of tasks. |
| version    | Parse version information from a local file. |
| xerrors    | Wrap errors with additional type-safe data using generics. Sub-packages for stacktraces, adding loggable context, and defined error classifications. |

## Contact Zircuit

We are happy to talk to you on [discord](https://discord.gg/zircuit)
