#!/bin/bash

# Health check script for finfocus-plugin-aws-public multi-region container
# Verifies that all 12 regional HTTP endpoints are responding

set -e

# Define ports
declare -a ports=(8001 8002 8003 8004 8005 8006 8007 8008 8009 8010 8011 8012)

echo "Running health check for all regional endpoints"

for port in "${ports[@]}"; do
    echo "Checking port ${port}..."

    # Try to connect to health endpoint
    if curl -f -s --max-time 5 "http://localhost:${port}/healthz" > /dev/null 2>&1; then
        echo "✓ Port ${port} is healthy"
    else
        echo "✗ Port ${port} is unhealthy"
        exit 1
    fi
done

echo "All endpoints are healthy"
exit 0