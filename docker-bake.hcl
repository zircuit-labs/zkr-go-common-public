# Build definitions for Zircuit zkr-go-common

# ----------- Input Variables and General Settings ------------

target "_go_tools" {
    # NOTE: Go compiler version is always latest from cgr.dev/chainguard/go:latest-dev
    #       However, it is the version set in go.mod that determines how things are built.
    args = {
        TPARSE_VERSION = "v0.18.0"
        GO_VULN_VERSION = "v1.1.4"
        STATIC_CHECK_VERSION = "2025.1"
        REVIVE_VERSION = "v1.12.0"
        GOLANGCI_VERSION = "v2.5.0"
    }
}

# ----------- Base targets ------------

target "go-builder" {
    inherits = ["_go_tools"]
    dockerfile = "./build/dockerfiles/go-builder.Dockerfile"
    output = ["type=cacheonly"]
}

# ----------- Intermediate targets ------------

target "go-copy-source" {
    context = "."
    contexts = {
        parent = "target:go-builder"
    }
    dockerfile = "./build/dockerfiles/go-copy-source.Dockerfile"
    output = ["type=cacheonly"]
}

# ----------- Test targets ------------

target "go-test" {
    contexts = {
        parent = "target:go-copy-source"
    }
    dockerfile = "./build/dockerfiles/go-test.Dockerfile"
    output = [
        "type=cacheonly",
        "type=local,dest=.coverage/",
    ]
}

# ----------- Linting targets ------------

group "linting" {
    targets = [
        "go-lint-govulncheck",
        "go-lint-staticcheck",
        "go-lint-revive",
        "go-lint-golangci",
    ]
}

target "go-lint-govulncheck" {
    args = {
        "LINT_COMMAND" = "govulncheck ./..."
    }
    contexts = {
        parent = "target:go-copy-source"
    }
    dockerfile = "./build/dockerfiles/go-lint.Dockerfile"
    output = ["type=cacheonly"]
}

target "go-lint-staticcheck" {
    args = {
        "LINT_COMMAND" = "staticcheck ./..."
    }
    contexts = {
        parent = "target:go-copy-source"
    }
    dockerfile = "./build/dockerfiles/go-lint.Dockerfile"
    output = ["type=cacheonly"]
}

target "go-lint-revive" {
    args = {
        "LINT_COMMAND" = "revive -set_exit_status -formatter stylish -config revive.toml ./..."
    }
    contexts = {
        parent = "target:go-copy-source"
    }
    dockerfile = "./build/dockerfiles/go-lint.Dockerfile"
    output = ["type=cacheonly"]
}
target "go-lint-golangci" {
    args = {
        "LINT_COMMAND" = "golangci-lint run --config=golangci.toml"
    }
    contexts = {
        parent = "target:go-copy-source"
    }
    dockerfile = "./build/dockerfiles/go-lint.Dockerfile"
    output = ["type=cacheonly"]
}


