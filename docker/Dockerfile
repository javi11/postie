# Multi-stage build for Postie Web
FROM oven/bun AS frontend-builder

# Set working directory for frontend
WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package.json frontend/bun.lock* ./

# Install dependencies
RUN bun install

# Copy frontend source
COPY frontend/ ./

# Build frontend
RUN bun run build

# Backend build stage
FROM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git gcc g++ musl-dev libstdc++ sqlite-dev

# Set working directory
WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./pkg ./pkg
COPY ./frontend/embed.go ./frontend/embed.go

# Copy built frontend from previous stage
COPY --from=frontend-builder /app/frontend/build ./frontend/build

# Build web binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o postie-web cmd/web/main.go

# Use Ubuntu as a parent image
FROM ghcr.io/linuxserver/baseimage-ubuntu:jammy

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

# Install necessary packages
RUN apt-get update -y && apt-get install --reinstall mime-support -y && apt-get install -y wget

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
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/status || exit 1

# Volume
VOLUME ["/config", "/watch", "/output"]

# Default command for web mode
CMD ["/app/postie-web"]

LABEL org.opencontainers.image.source="https://github.com/javi11/postie"
LABEL build_version="version: Build-date:- ${VERSION}  ${BUILD_DATE}"
LABEL maintainer="javi11"
