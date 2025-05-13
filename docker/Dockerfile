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
RUN apt-get update -y && apt-get install --reinstall mime-support -y

# Create the service directory and the run script
RUN mkdir -p /etc/services.d/postie && \
    echo '#!/usr/bin/with-contenv bash\n\
    groupmod -o -g "$PGID" abc\n\
    usermod -o -u "$PUID" abc\n\
    chown -R abc:abc /app /config\n\
    exec s6-setuidgid abc /app/postie watch --config /config/config.yaml --dir /watch --output-folder /output\n\
    ' > /etc/services.d/postie/run && \
    chmod +x /etc/services.d/postie/run

# Binary
COPY postie /app/postie

# Volume
VOLUME ["/config", "/watch", "/output"]

# Add root files
CMD ["/app/postie" , "watch" , "--config", "/config/config.yaml" , "--dir" , "/watch" , "--output-folder" , "/output"]

LABEL org.opencontainers.image.source="https://github.com/javi11/usenet-drive"
LABEL build_version="version: Build-date:- ${VERSION}  ${BUILD_DATE}"
LABEL maintainer="javi11"
