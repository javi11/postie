# Multi-stage build for Postie Web - Multi-platform optimized
# Use buildkit syntax for advanced features
# syntax=docker/dockerfile:1.7

# Frontend build stage
FROM --platform=$BUILDPLATFORM oven/bun:1 AS frontend-builder

# Set working directory for frontend
WORKDIR /app/frontend

# Copy only dependency manifests first to leverage caching for bun install
COPY frontend/bun.lock frontend/package.json ./

# Install dependencies with cache mount
RUN --mount=type=cache,target=/root/.bun/install/cache,sharing=locked \
    bun install --frozen-lockfile

# Copy the rest of the frontend sources and build
COPY frontend/ ./
RUN bun run build

# Backend build stage with multi-arch support
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-builder

# Build arguments for multi-platform support
ARG TARGETOS
ARG TARGETARCH

# Install build dependencies
RUN apk add --no-cache git gcc g++ musl-dev libstdc++ sqlite-dev

# Set working directory
WORKDIR /app

# Copy go module files first for better caching
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

# Copy source code
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./pkg ./pkg
COPY ./frontend/embed.go ./frontend/embed.go

# Copy built frontend from previous stage
COPY --from=frontend-builder /app/frontend/build ./frontend/build

# Build web binary with cache mount and cross-compilation support
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -a -ldflags '-linkmode external -extldflags "-static"' -o postie-web cmd/web/main.go

# Final stage - use architecture-specific base image with conditional selection
ARG TARGETARCH
FROM ghcr.io/linuxserver/baseimage-ubuntu:${TARGETARCH:-amd64}-jammy

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

# Create the entrypoint script for web mode (optimized for both architectures)
RUN mkdir -p /etc/services.d/postie-web && \
    echo '#!/usr/bin/with-contenv bash\n\
groupmod -o -g "$PGID" abc 2>/dev/null || true\n\
usermod -o -u "$PUID" abc 2>/dev/null || true\n\
chown -R abc:abc /app /config /watch /output 2>/dev/null || true\n\
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

# Volume
VOLUME ["/config", "/watch", "/output"]

# Default command for web mode
CMD ["/app/postie-web"]

LABEL org.opencontainers.image.source="https://github.com/javi11/postie"
LABEL build_version="version: Build-date:- ${VERSION}  ${BUILD_DATE}"
LABEL maintainer="javi11"
