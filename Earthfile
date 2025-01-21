# Earthfile
VERSION 0.8

# version of Go compiler to use
# remember to update both SHA versions as well
# when updating major version, also be sure to look at the .github/worflows files
ARG --global GO_VERSION=1.23.1
ARG --global GO_SHA_AMD64=49bbb517cfa9eee677e1e7897f7cf9cfdbcf49e05f61984a2789136de359f9bd
ARG --global GO_SHA_ARM64=faec7f7f8ae53fda0f3d408f52182d942cc89ef5b7d3d9f23ff117437d4b2d2f

# ------------------------- Base Images -----------------------------

docker-base:
    FROM cgr.dev/chainguard/wolfi-base
    RUN apk add --no-cache wget jq curl

INSTALL_GO_CMD:
    FUNCTION
    IF [ "$(uname -m)" = "x86_64" ]
        ENV GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
        ENV GO_URL="https://golang.org/dl/${GO_TAR}"
        ENV GO_SHA="${GO_SHA_AMD64}"
    ELSE
        ENV GO_TAR="go${GO_VERSION}.linux-arm64.tar.gz"
        ENV GO_URL="https://golang.org/dl/${GO_TAR}"
        ENV GO_SHA="${GO_SHA_ARM64}"
    END
    WORKDIR /Downloads
    RUN wget -nv "${GO_URL}"
    RUN echo "${GO_SHA} ${GO_TAR}" | sha256sum -c
    RUN tar -C /usr/ -xzf "${GO_TAR}"
    RUN rm "${GO_TAR}"
    IF [ -d "/usr/go/bin" ]
        ENV PATH=$PATH:/usr/go/bin
        ENV GOBIN=/usr/bin
    ELSE
        ENV PATH=$PATH:/usr/local/go/bin
        ENV GOBIN=/usr/local/go/bin
    END
    RUN go version

go-builder:
    FROM +docker-base
    DO +INSTALL_GO_CMD
    # Add build/test tools
    RUN go install github.com/mfridman/tparse@latest
    RUN go install golang.org/x/vuln/cmd/govulncheck@latest
    RUN go install honnef.co/go/tools/cmd/staticcheck@latest
    RUN go install github.com/mgechev/revive@latest
    RUN go install github.com/butuzov/mirror/cmd/mirror@latest
    RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $GOBIN v1.60.3

# ------------------------- Intermediate Images -----------------------------

go-build-dependencies:
    FROM +go-builder
    # Copy only the go mod files and download/verify.
    WORKDIR /src/
    COPY ./go.mod /src
    COPY ./go.sum /src
    RUN go mod download
    RUN go mod verify

go-build-copy-source:
    FROM +go-build-dependencies
    # Copy all the code
    # This is a separate step to avoid cache-busting when code changes are made that don't affect the dependancies.
    COPY . /src

# ------------------------- Testing -----------------------------

go-test-coverage-unit:
    FROM +go-build-copy-source
    WORKDIR /src/
    RUN go test -json -shuffle=on -covermode=atomic -coverprofile=/coverage.unit ./... | tparse -all -progress

# ------------------------- Linting -----------------------------

# Run govulncheck
go-lint-govulncheck:
    FROM +go-build-copy-source
    WORKDIR /src
    RUN govulncheck ./...

# Run staticcheck - is part of golangci-lint but unfortunately not the same thing
go-lint-staticcheck:
    FROM +go-build-copy-source
    WORKDIR /src
    RUN staticcheck ./...

# Run revive - is part of golangci-lint but unfortunately difficult to configure there
go-lint-revive:
    FROM +go-build-copy-source
    WORKDIR /src
    RUN revive -set_exit_status -formatter stylish -config /src/revive.toml ./...

# Mirror is not yet part of other linters
# This looks for redundant conversions between types
go-lint-mirror:
    FROM +go-build-copy-source
    WORKDIR /src
    RUN mirror ./...

# Run golangci-lint - massive set of checks; highly configurable.
go-lint-golangci:
    FROM +go-build-copy-source
    WORKDIR /src
    RUN golangci-lint run --config=.golangci.toml
