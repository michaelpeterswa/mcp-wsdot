# go.dockerfile
#
# Builds the Go application and installs its binary (or binaries) into
# /usr/local/bin of the final image.

# STAGE 1
# Build the executable(s).
#
# Pinned to the native BUILDPLATFORM so the Go toolchain runs without emulation
# and cross-compiles to TARGETARCH via GOARCH. This is dramatically faster than
# running the compiler under QEMU for a foreign architecture.
FROM --platform=$BUILDPLATFORM golang:1-bookworm AS stage1

ARG VERSION=dev
ARG PKG_VER_PATH=github.com/michaelpeterswa/mcp-wsdot/internal/config.AppVersion

WORKDIR /var/build/go
ARG TARGETARCH
ENV GOARCH=$TARGETARCH
ENV CGO_ENABLED=0

# Download modules in their own layer so they are only refetched when
# go.mod/go.sum change, not on every source edit. The module and build caches
# persist across builds via BuildKit cache mounts.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -v -trimpath \
        -ldflags "-s -w -X ${PKG_VER_PATH}=${VERSION}" \
        -o /var/build/bin/ ./...

# STAGE 2
# Prepare the base image.
FROM debian:bookworm AS stage2

# hadolint ignore=DL3008
RUN apt-get update --fix-missing && \
    apt-get install -yqq --no-install-recommends \
        ca-certificates \
        curl \
        tzdata \
        && \
    apt-get autoclean -yqq && \
    apt-get clean -yqq && \
    rm -rf /var/lib/apt/lists/*

# STAGE 3
# Construct the final image.
# Can be merged into stage 2, but can make better use of caches when split.
FROM stage2 AS stage3

ARG VERSION=dev

LABEL org.opencontainers.image.source="https://github.com/michaelpeterswa/mcp-wsdot" \
      org.opencontainers.image.description="MCP server for the WSDOT API" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}"

# Copies the mcp-wsdot binary built from the ./cmd/ directory.
COPY --from=stage1 /var/build/bin/* /usr/local/bin/

# streamablehttp/sse listen port; override with HTTP_PORT.
EXPOSE 8080
# prometheus metrics port; override with METRICS_PORT.
EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/mcp-wsdot"]
