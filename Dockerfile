# -=-=-=-=-=-=- Compile Image -=-=-=-=-=-=-

ARG VERSION=unset
ARG VER_PATH=github.com/michaelpeterswa/go-mcp-template/internal/config.AppVersion

FROM golang:1 AS stage-compile

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./... && CGO_ENABLED=0 GOOS=linux go build ./cmd/go-mcp-template

# -=-=-=-=- Final Distroless Image -=-=-=-=-

# hadolint ignore=DL3007
FROM gcr.io/distroless/static-debian12:latest AS stage-final

COPY --from=stage-compile /go/src/app/go-mcp-template /
CMD ["/go-mcp-template"]