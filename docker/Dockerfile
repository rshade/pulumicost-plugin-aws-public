# Multi-Region Docker Image for finfocus-plugin-aws-public
FROM alpine:3.19

# Install dependencies
RUN apk add --no-cache \
    ca-certificates=20241121-r1 \
    curl=8.5.0-r0 \
    tini=0.19.0-r2

# Set working directory
WORKDIR /app

# Define build arguments
ARG VERSION=v0.1.0
ENV VERSION=${VERSION}

# Define regions
ENV REGIONS="us-east-1 us-west-2 eu-west-1 ap-southeast-1 ap-southeast-2 ap-northeast-1 ap-south-1 ca-central-1 sa-east-1 us-gov-west-1 us-gov-east-1 us-west-1"

# Download and extract regional binaries
RUN set -ex; \
    arch=$(uname -m); \
    case "$arch" in \
        x86_64) release_arch="x86_64" ;; \
        aarch64) release_arch="arm64" ;; \
        *) echo "Unsupported architecture: ${arch}"; exit 1 ;; \
    esac; \
    tarball="finfocus-plugin-aws-public_${VERSION}_Linux_${release_arch}.tar.gz"; \
    curl -L -o "/tmp/${tarball}" \
        "https://github.com/rshade/finfocus-plugin-aws-public/releases/download/${VERSION}/${tarball}"; \
    tar -xzf "/tmp/${tarball}" -C /tmp; \
    rm "/tmp/${tarball}"; \
    for region in $REGIONS; do \
        cp "/tmp/finfocus-plugin-aws-public-${region}" \
            "/usr/local/bin/finfocus-plugin-aws-public-${region}"; \
    done; \
    rm -f /tmp/finfocus-plugin-aws-public-*

# Copy metrics aggregator source and build it
COPY go.mod go.sum /tmp/metrics-aggregator/
COPY cmd/metrics-aggregator/ /tmp/metrics-aggregator/
WORKDIR /tmp/metrics-aggregator
RUN set -ex; \
    apk add --no-cache go; \
    go build -o /usr/local/bin/metrics-aggregator .; \
    apk del go; \
    rm -rf /tmp/metrics-aggregator

WORKDIR /app

# Copy scripts
COPY docker/entrypoint.sh /entrypoint.sh
COPY docker/healthcheck.sh /healthcheck.sh

# Make scripts executable
RUN chmod +x /entrypoint.sh /healthcheck.sh

# Create non-root user
RUN addgroup -g 65532 -S plugin && \
    adduser -u 65532 -S plugin -G plugin

# Expose ports
EXPOSE 8001 8002 8003 8004 8005 8006 8007 8008 8009 8010 8011 8012 9090

# Set default environment variables
ENV FINFOCUS_PLUGIN_WEB_ENABLED=true
ENV FINFOCUS_PLUGIN_HEALTH_ENDPOINT=true

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD /healthcheck.sh

# Use tini as init process
ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]

# Switch to non-root user
USER plugin