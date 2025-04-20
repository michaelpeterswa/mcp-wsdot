# -=-=-=-=-=-=- Compile Image -=-=-=-=-=-=-

ARG VERSION=unset
ARG VER_PATH=github.com/michaelpeterswa/mcp-wsdot/internal/config.AppVersion

FROM golang:1 AS stage-compile

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./... && CGO_ENABLED=0 GOOS=linux go build ./cmd/mcp-wsdot

# -=-=-=-=- Final Distroless Image -=-=-=-=-

# hadolint ignore=DL3007
FROM gcr.io/distroless/static-debian12:latest AS stage-final

COPY --from=stage-compile /go/src/app/mcp-wsdot /
CMD ["/mcp-wsdot"]