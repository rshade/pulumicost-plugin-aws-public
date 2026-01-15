# Multi-Region Docker Image for finfocus-plugin-aws-public
FROM alpine:3.19

# Install dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    tini

# Set working directory
WORKDIR /app

# Define build arguments
ARG VERSION=v0.1.0
ENV VERSION=${VERSION}

# Define regions
ENV REGIONS="us-east-1 us-west-2 eu-west-1 ap-southeast-1 ap-southeast-2 ap-northeast-1 ap-south-1 ca-central-1 sa-east-1 us-gov-west-1 us-gov-east-1 us-west-1"

# Download and extract regional binaries
RUN set -ex; \
    for region in $REGIONS; do \
        echo "Downloading binary for ${region}"; \
        curl -L -o "/tmp/finfocus-plugin-aws-public-${region}.tar.gz" \
            "https://github.com/rshade/finfocus-plugin-aws-public/releases/download/${VERSION}/finfocus-plugin-aws-public-${region}.tar.gz" || exit 1; \
        tar -xzf "/tmp/finfocus-plugin-aws-public-${region}.tar.gz" -C /usr/local/bin/; \
        rm "/tmp/finfocus-plugin-aws-public-${region}.tar.gz"; \
    done

# Copy metrics aggregator source and build it
COPY cmd/metrics-aggregator/ /tmp/metrics-aggregator/
RUN set -ex; \
    apk add --no-cache go; \
    cd /tmp/metrics-aggregator; \
    go build -o /usr/local/bin/metrics-aggregator .; \
    apk del go; \
    rm -rf /tmp/metrics-aggregator

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