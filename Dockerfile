# syntax=docker/dockerfile:1.6
#
# Multi-stage build for the ACP server.
# - Builder stage uses the pinned Go toolchain.
# - Runtime stage is distroless static for minimal attack surface.

ARG GO_VERSION=1.26.3
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary.
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/acp-server ./cmd/acp-server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/acp-server /acp-server

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/acp-server"]
