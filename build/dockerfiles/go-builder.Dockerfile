# Clean base image
FROM cgr.dev/chainguard/go:latest-dev

ARG TPARSE_VERSION
ARG GO_VULN_VERSION
ARG STATIC_CHECK_VERSION
ARG REVIVE_VERSION
ARG GOLANGCI_VERSION

# Install dependencies / non-standard helpful dev tools
RUN apk add --no-cache ripgrep jq

# Set Path/Go environment variables including go cache and module cache directories for cache mounting
ENV GOBIN=/usr/bin
ENV GOCACHE=/go-cache
ENV GOMODCACHE=/gomod-cache
ENV GOPRIVATE=github.com/zircuit-labs/*

# Add build/test tools
# Tools should be installed before code is copied to the image
# to take advantage of Docker's caching mechanism
ENV CGO_ENABLED=0
RUN go install github.com/mfridman/tparse@${TPARSE_VERSION}
RUN go install golang.org/x/vuln/cmd/govulncheck@${GO_VULN_VERSION}
RUN go install honnef.co/go/tools/cmd/staticcheck@${STATIC_CHECK_VERSION}
RUN go install github.com/mgechev/revive@${REVIVE_VERSION}
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION}

# Enable CGO only after installing the tools, as `golangci-lint` in particular has issues on arm64 with CGO
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-fPIC -O -D__BLST_PORTABLE__"
ENV CGO_CFLAGS_ALLOW="-O -D__BLST_PORTABLE__"
ENV CGO_LDFLAGS="-s -w -Wl,-z,stack-size=0x800000"
