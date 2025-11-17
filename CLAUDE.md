# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `pulumicost-plugin-aws-public`, a fallback PulumiCost plugin that estimates AWS resource costs using public AWS on-demand pricing data, without requiring CUR/Cost Explorer/Vantage data access. The plugin is designed to be called as an external binary by PulumiCost core.

## Architecture

### Plugin Protocol
- The plugin reads AWS resource data (JSON) from **stdin**
- It writes a JSON envelope with estimated costs to **stdout**
- Exit code 0 for success (`status: "ok"`), non-zero for errors (`status: "error"`)
- The plugin is **stateless** - each invocation is independent

### Region-Specific Binaries
- **One binary per AWS region** using GoReleaser with build tags
- Binary naming: `pulumicost-plugin-aws-public-<region>` (e.g., `pulumicost-plugin-aws-public-us-east-1`)
- Each binary embeds only its region's pricing data via `//go:embed`
- Build tag mapping:
  - `us-east-1` → `region_use1`
  - `us-west-2` → `region_usw2`
  - `eu-west-1` → `region_euw1`

### Embedded Pricing Data
- At build time: `tools/generate-pricing` fetches/trims AWS public pricing
- Output: `data/aws_pricing_<region>.json` files
- These files are embedded into binaries using `//go:embed` in region-specific files under `internal/pricing/`
- The pricing client parses embedded JSON once using `sync.Once` and builds lookup indexes

### Service Support (v1)
- **Fully implemented**: EC2 instances, EBS volumes
- **Stubbed/Partial**: S3, Lambda, RDS, DynamoDB (emit MonthlyCost=0 with low confidence)

## Directory Structure

```
cmd/
  pulumicost-plugin-aws-public/     # CLI entrypoint
    main.go
internal/
  plugin/
    types.go        # PluginResponse, StackEstimate, ResourceCostEstimate
    plugin.go       # Plugin interface and wrapper
    estimator.go    # AWSEstimator implementation (EC2/EBS logic)
  pricing/
    client.go       # Pricing client with lookup methods
    embed_*.go      # Region-specific embedded pricing (build-tagged)
  config/
    config.go       # Configuration (currency, discount factor)
tools/
  generate-pricing/
    main.go         # Build-time tool to fetch/trim AWS pricing
data/
  aws_pricing_*.json  # Generated pricing files (not in git)
```

## Common Commands

### Building
```bash
# Standard build (no region tags, uses default fallback)
go build ./...

# Build with specific region tag
go build -tags region_use1 -o pulumicost-plugin-aws-public-us-east-1 ./cmd/pulumicost-plugin-aws-public

# Build all region binaries using GoReleaser
goreleaser build --snapshot --clean
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/plugin
go test ./internal/pricing
```

### Generating Pricing Data
```bash
# Generate dummy pricing data for development
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy

# (Future) Real AWS pricing fetch
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data
```

### Running the Plugin
```bash
# Pipe a StackInput JSON to the plugin
cat testdata/input_ec2_ebs.json | ./pulumicost-plugin-aws-public-us-east-1

# Or with stdin redirection
./pulumicost-plugin-aws-public-us-east-1 < testdata/input_ec2_ebs.json
```

## Key Types and Protocols

### PluginResponse Envelope
All plugin output uses this structure:
```go
type PluginResponse struct {
    Version  int             `json:"version"`     // Always 1
    Status   string          `json:"status"`      // "ok" | "error"
    Result   *StackEstimate  `json:"result,omitempty"`
    Error    *PluginError    `json:"error,omitempty"`
    Warnings []PluginWarning `json:"warnings,omitempty"`
}
```

### Error Codes
- `INVALID_INPUT`: stdin JSON parsing failed
- `PRICING_INIT_FAILED`: embedded pricing data could not be loaded
- `UNSUPPORTED_REGION`: resources are in a different region than the plugin binary
  - This error includes `meta.pluginRegion` and `meta.requiredRegion` to help PulumiCost core fetch the correct binary
- `NOT_IMPLEMENTED`: placeholder for unimplemented functionality

### Resource Input Format
Resources come from PulumiCost core as:
```go
type ResourceInput struct {
    URN        string                 `json:"urn"`
    Provider   string                 `json:"provider"`     // "aws"
    Type       string                 `json:"type"`         // e.g. "aws:ec2/instance:Instance"
    Name       string                 `json:"name"`
    Region     string                 `json:"region"`
    Properties map[string]any         `json:"properties"`
}
```

## Estimation Logic

### EC2 Instances
- Resource type: `aws:ec2/instance:Instance`
- Required property: `instanceType`
- Assumptions (hardcoded for v1):
  - `operatingSystem = "Linux"`
  - `tenancy = "Shared"`
  - `hoursPerMonth = 730` (24×7 on-demand)
- Confidence: "high" if pricing found, "none" otherwise

### EBS Volumes
- Resource type: `aws:ebs/volume:Volume`
- Required properties: `volumeType` (e.g., `gp2`, `gp3`), `size` (GB)
- Default size: 8 GB if missing (marked in assumptions)
- Confidence: "high" if all data available and pricing found, "medium" if size assumed, "none" if no pricing

### Region Mismatch Handling
- If **all resources** share one region different from plugin binary region → return `UNSUPPORTED_REGION` error
- If **some resources** are in different regions → skip those, add warnings, continue with matching resources

## Development Notes

### Adding New AWS Services
1. Add service-specific estimation logic in `internal/plugin/estimator.go`
2. Create a helper function like `estimateEC2()` or `estimateEBS()`
3. Update the main `Estimate()` loop to call the new helper
4. Extend `tools/generate-pricing` to fetch pricing for the new service
5. Update `internal/pricing/client.go` with lookup methods for the new service

### Working with Build Tags
- Region-specific files use build tags like `//go:build region_use1`
- The fallback file uses negation: `//go:build !region_use1 && !region_usw2 && !region_euw1`
- Always ensure exactly one embed file is selected at build time

### Testing with Dummy Pricing
- Use `--dummy` flag with `tools/generate-pricing` to create minimal test pricing data
- This allows development/testing without AWS API access or credentials
- Dummy data includes only essential instance types (e.g., `t3.micro`) and volume types

### Logging
- **Never** log to stdout - stdout is reserved for JSON protocol
- Use stderr with prefix `[pulumicost-plugin-aws-public]` for debug/diagnostic messages
- Keep logging minimal by default

## Release Process

### Using GoReleaser
```bash
# Test release build locally
goreleaser build --snapshot --clean

# Create actual release
goreleaser release --clean
```

### Before Hooks
The `.goreleaser.yaml` runs:
```bash
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy
```

This ensures pricing files exist before embedding. For production, remove `--dummy` and implement real AWS pricing fetch.

## Configuration (v1)

Current configuration is minimal:
- Currency: `USD` (hardcoded default)
- Account discount factor: `1.0` (no discount)

Future versions will support:
- Environment variables or flags for configuration
- Custom discount rates
- Different EC2 tenancy models
- Spot/Reserved instance pricing
