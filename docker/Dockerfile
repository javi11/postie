# Multi-stage build for Postie Web
# Use buildkit syntax for advanced features
# syntax=docker/dockerfile:1.4

# Frontend build stage
FROM --platform=$BUILDPLATFORM oven/bun:1 AS frontend-builder

# Set working directory for frontend
WORKDIR /app/frontend

# Copy frontend source (single step to avoid missing paths in GoReleaser context)
COPY frontend/ ./

# Install dependencies (cached layer) and build
RUN --mount=type=cache,target=/root/.bun \
    bun install --frozen-lockfile
RUN bun run build

# Backend build stage with multi-arch support
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git gcc g++ musl-dev libstdc++ sqlite-dev

# Set working directory
WORKDIR /app

# Copy go module files first for better caching
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./pkg ./pkg
COPY ./frontend/embed.go ./frontend/embed.go

# Copy built frontend from previous stage
COPY --from=frontend-builder /app/frontend/build ./frontend/build

# Build arguments for cross-compilation
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Build web binary with caching
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -a -ldflags '-linkmode external -extldflags "-static"' \
    -o postie-web cmd/web/main.go

# Final stage - use architecture-specific base image
FROM ghcr.io/linuxserver/baseimage-ubuntu:jammy AS final

ARG DEBIAN_FRONTEND="noninteractive"
ARG BUILD_DATE
ARG VERSION
ARG PUID=1000
ARG PGID=1000

# Set up environment variables for PUID and PGID
ENV PUID=${PUID}
ENV PGID=${PGID}

# Set the working directory inside the container
WORKDIR /app

# Install necessary packages with cache mount for faster builds
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    apt-get update -y && \
    apt-get install --reinstall mime-support -y && \
    apt-get install -y wget && \
    rm -rf /var/lib/apt/lists/*

# Create the entrypoint script for web mode
RUN mkdir -p /etc/services.d/postie-web && \
    echo '#!/usr/bin/with-contenv bash\n\
groupmod -o -g "$PGID" abc\n\
usermod -o -u "$PUID" abc\n\
chown -R abc:abc /app /config\n\
exec s6-setuidgid abc /app/postie-web\n\
' > /etc/services.d/postie-web/run && \
    chmod +x /etc/services.d/postie-web/run

# Copy web binary from builder
COPY --from=backend-builder /app/postie-web /app/postie-web

# Create necessary directories
RUN mkdir -p /config /watch /output

# Set environment variables
ENV PORT=8080
ENV HOST=0.0.0.0

# Expose port for web interface
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/live || exit 1

# Volume
VOLUME ["/config", "/watch", "/output"]

# Default command for web mode
CMD ["/app/postie-web"]

LABEL org.opencontainers.image.source="https://github.com/javi11/postie"
LABEL build_version="version: Build-date:- ${VERSION}  ${BUILD_DATE}"
LABEL maintainer="javi11"
