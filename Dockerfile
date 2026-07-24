# -=-=-=-=-=-=- Compile Image -=-=-=-=-=-=-

ARG VERSION=unset
ARG VER_PATH=github.com/michaelpeterswa/mcp-wsdot/internal/config.AppVersion

FROM golang:1.25 AS stage-compile

ARG VERSION
ARG VER_PATH
ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/app

# Prime the module cache separately so it is only rebuilt when go.mod/go.sum
# change, not on every source edit.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X ${VER_PATH}=${VERSION}" -o /mcp-wsdot ./cmd/mcp-wsdot

# -=-=-=-=- Final Distroless Image -=-=-=-=-

# hadolint ignore=DL3007
FROM gcr.io/distroless/static-debian12:nonroot AS stage-final

ARG VERSION

LABEL org.opencontainers.image.source="https://github.com/michaelpeterswa/mcp-wsdot" \
      org.opencontainers.image.description="MCP server for the WSDOT API" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}"

COPY --from=stage-compile /mcp-wsdot /mcp-wsdot

# streamablehttp/sse listen port; override with HTTP_PORT.
EXPOSE 8080
# prometheus metrics port; override with METRICS_PORT.
EXPOSE 8081

# nonroot (uid 65532) is provided by the distroless base image.
USER nonroot:nonroot

ENTRYPOINT ["/mcp-wsdot"]
