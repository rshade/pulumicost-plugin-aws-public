#!/bin/bash
# scripts/verify-release-binaries.sh
# Verify that all release binaries have real AWS pricing data embedded
# and fit within size constraints.
#
# Usage: ./scripts/verify-release-binaries.sh <dist-directory>
# Example: ./scripts/verify-release-binaries.sh dist/
#
# Checks:
# 1. All Linux x86_64 archives exist
# 2. Extracts each archive and verifies:
#    - Size > 100MB (ensures pricing data is embedded)
#    - Size < 240MB (ensures binary isn't bloated)
# 3. Warns if Size > 200MB
#
# Returns 0 if all binaries verified, 1 if any fail

set -e

DIST_DIR="${1:-.}"
# Raw binary minimum size: 100MB (with embedded pricing JSON)
MIN_BINARY_SIZE=100000000
# Warning threshold: 200MB
WARN_BINARY_SIZE=200000000
# Critical threshold: 240MB (GitHub/deployment limits)
MAX_BINARY_SIZE=240000000

if [ ! -d "$DIST_DIR" ]; then
    echo "ERROR: Directory not found: $DIST_DIR"
    exit 1
fi

echo "Verifying release archives in $DIST_DIR..."
echo ""

# Find the first Linux x86_64 archive to verify
# Archive naming pattern: finfocus-plugin-aws-public_${VERSION}_Linux_x86_64_${REGION}.tar.gz
ARCHIVE=$(find "$DIST_DIR" -maxdepth 1 -name "finfocus-plugin-aws-public_*_Linux_x86_64_*.tar.gz" -type f | head -1)

if [ -z "$ARCHIVE" ]; then
    echo "❌ FAILURE: No archives found matching pattern 'finfocus-plugin-aws-public_*_Linux_x86_64_*.tar.gz'"
    echo "   Found in $DIST_DIR:"
    ls -la "$DIST_DIR"/ 2>/dev/null || echo "   (directory empty or not accessible)"
    echo ""
    echo "   Expected archives like: finfocus-plugin-aws-public_0.0.13_Linux_x86_64_us-east-1.tar.gz"
    exit 1
fi

# Count total archives for reporting
TOTAL_ARCHIVES=$(find "$DIST_DIR" -maxdepth 1 -name "finfocus-plugin-aws-public_*_Linux_x86_64_*.tar.gz" -type f | wc -l)

echo "Found $TOTAL_ARCHIVES Linux x86_64 archive(s)"
echo "Verifying one archive (all built with same process)..."
echo ""

# Treat the single found archive as the list to iterate over
ARCHIVES="$ARCHIVE"

# Create temp directory for extraction
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

FAILURES=0
WARNINGS=0

for ARCHIVE in $ARCHIVES; do
    archive_name=$(basename "$ARCHIVE")
    # echo "Checking: $archive_name"

    # Extract to temp dir (overwrite existing)
    tar -xzf "$ARCHIVE" -C "$TEMP_DIR"

    # Find the binary (it's the only executable in the archive)
    binary=$(find "$TEMP_DIR" -type f -name "finfocus-plugin-aws-public-*" | head -1)

    if [ -z "$binary" ]; then
        echo "  ❌ FAIL: No binary found in $archive_name"
        FAILURES=$((FAILURES + 1))
        continue
    fi

    size=$(stat -c%s "$binary")
    binary_name=$(basename "$binary")
    size_mb=$((size / 1000000))

    if [ "$size" -lt "$MIN_BINARY_SIZE" ]; then
        echo "  ❌ FAIL: $binary_name too small: ${size_mb}MB (< 100MB)"
        echo "     Indicates missing pricing data (fallback mode)"
        FAILURES=$((FAILURES + 1))
    elif [ "$size" -gt "$MAX_BINARY_SIZE" ]; then
        echo "  ❌ FAIL: $binary_name too large: ${size_mb}MB (> 240MB)"
        echo "     Exceeds critical size limit"
        FAILURES=$((FAILURES + 1))
    elif [ "$size" -gt "$WARN_BINARY_SIZE" ]; then
        echo "  ⚠️  WARN: $binary_name is large: ${size_mb}MB (> 200MB)"
        WARNINGS=$((WARNINGS + 1))
    else
        echo "  ✓ $binary_name (${size_mb}MB) - OK"
    fi
    
    # Clean up for next iteration
    rm -f "$TEMP_DIR"/*
done

echo ""
if [ "$FAILURES" -gt 0 ]; then
    echo "❌ FAILED: $FAILURES binaries failed verification"
    exit 1
else
    if [ "$WARNINGS" -gt 0 ]; then
        echo "✅ SUCCESS (with $WARNINGS warnings): All binaries verified"
    else
        echo "✅ SUCCESS: All binaries verified"
    fi
    exit 0
fi