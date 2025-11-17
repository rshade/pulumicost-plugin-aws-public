# PulumiCost AWS Public Plugin – Prompt Pack Overview

You are OpenCode v0.15.3 using the GrokZeroFree model.
Your job is to implement the `pulumicost-plugin-aws-public` Go plugin according to the design below.

## Context

- Repo: `https://github.com/rshade/pulumicost-plugin-aws-public`
- Language: Go
- Purpose: A **fallback cost plugin** for PulumiCost that estimates AWS resource costs using **public AWS on‑demand pricing**, without needing CUR/Cost Explorer/Vantage data.
- Architecture:
  - The PulumiCost **core** calls external plugins as binaries.
  - This plugin reads a stack’s AWS resources (JSON) from stdin and writes back a JSON envelope with estimated costs.
- Key design choices (already agreed):
  1. Use **embedded, trimmed JSON** pricing per region via `//go:embed` (Option A).
  2. Use **GoReleaser** to build **one binary per region**, each with only that region’s pricing.
  3. Binary naming convention:
     - `pulumicost-plugin-aws-public-<region>`
     - e.g. `pulumicost-plugin-aws-public-us-east-1`
  4. For now, **single-region** per binary; multi-region stacks are handled by having core call multiple region binaries.
  5. Error protocol:
     - Plugin returns a **JSON envelope** with `status: "ok"` or `status: "error"`.
     - On mismatched region, error code: `UNSUPPORTED_REGION` plus metadata telling core which region is required.
     - Core is responsible for **downloading/using** the correct region binary, but we still add the error in the plugin so core can act on it later.

## Services to support in v1

Start small but useful:

- **Required in v1**
  - EC2 instances
  - EBS volumes
- Nice to scaffold / stub (but can be partially unimplemented):
  - S3 buckets
  - Lambda
  - RDS
  - DynamoDB

For unimplemented services, plugin should emit `MonthlyCost=0` with low confidence and a clear assumption / warning.

## Pricing data

- Source: AWS public price list JSON endpoints.
- We **do not** hit these at runtime.
- Instead:
  - A small Go tool in `tools/generate-pricing` calls AWS pricing APIs at build/release time.
  - It **trims** the data to only the fields/services we need.
  - Writes per-region JSON files into `data/aws_pricing_<region>.json`.
- The plugin embeds one such JSON per region using `//go:embed`.

## Build tags per region

- Each region has its own file under `internal/pricing`:

  ```go
  // internal/pricing/embed_use1.go
  //go:build region_use1

  package pricing

  import _ "embed"

  //go:embed ../../data/aws_pricing_us-east-1.json
  var rawPricingJSON []byte

  const Region = "us-east-1"
  ```

- Example mapping:
  - `us-east-1` → build tag `region_use1`
  - `us-west-2` → build tag `region_usw2`
  - `eu-west-1` → build tag `region_euw1`

## Plugin protocol

The plugin **always** writes a JSON envelope like:

```jsonc
{
  "version": 1,
  "status": "ok", // or "error"
  "result": {
    "resources": [/* ResourceCostEstimate */],
    "totalMonthly": 123.45,
    "totalHourly": 0.17
  },
  "error": null,
  "warnings": []
}
```

When there is a region mismatch (e.g. all resources are in `us-west-2` but this binary is compiled for `us-east-1`), we return:

```jsonc
{
  "version": 1,
  "status": "error",
  "error": {
    "code": "UNSUPPORTED_REGION",
    "message": "This aws-public plugin is compiled for us-east-1 but resources are in us-west-2.",
    "meta": {
      "pluginRegion": "us-east-1",
      "requiredRegion": "us-west-2"
    }
  }
}
```

Exit code should be **non-zero** when `status = "error"`.

## Envelope types

Define shared types (these can live in `internal/plugin/types.go`):

```go
type PluginResponse struct {
    Version  int             `json:"version"`
    Status   string          `json:"status"` // "ok" | "error"
    Result   *StackEstimate  `json:"result,omitempty"`
    Error    *PluginError    `json:"error,omitempty"`
    Warnings []PluginWarning `json:"warnings,omitempty"`
}

type PluginError struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Meta    map[string]any         `json:"meta,omitempty"`
}

type PluginWarning struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Meta    map[string]any         `json:"meta,omitempty"`
}
```

`StackEstimate` and `ResourceCostEstimate` should be designed so they are compatible with the Vantage plugin and PulumiCost core:

```go
type StackEstimate struct {
    Resources   []ResourceCostEstimate `json:"resources"`
    TotalMonthly float64               `json:"totalMonthly"`
    TotalHourly  float64               `json:"totalHourly"`
    Currency     string                `json:"currency"`
}

type ResourceCostEstimate struct {
    URN          string          `json:"urn"`
    Service      string          `json:"service"`
    ResourceType string          `json:"resourceType"`
    Region       string          `json:"region"`
    MonthlyCost  float64         `json:"monthlyCost"`
    HourlyCost   float64         `json:"hourlyCost"`
    Currency     string          `json:"currency"`
    Confidence   string          `json:"confidence"` // "high" | "medium" | "low" | "none"
    LineItems    []LineItemCost  `json:"lineItems"`
    Assumptions  []Assumption    `json:"assumptions"`
}

type LineItemCost struct {
    Description string  `json:"description"`
    Unit        string  `json:"unit"`
    Quantity    float64 `json:"quantity"`
    Rate        float64 `json:"rate"`
    Cost        float64 `json:"cost"`
}

type Assumption struct {
    Key         string `json:"key"`
    Description string `json:"description"`
    Value       string `json:"value"`
}
```

## Configuration & usage assumptions

- The plugin will eventually support configuration (profiles, discounts, etc) via env or flags, but for v1, keep it minimal:
  - Currency: default `USD`
  - Account discount factor: default `1.0` (no discount)
  - EC2: assume 730 hours/month (24x7 on-demand).
  - EBS: cost = size (GB) * `rate_per_gb_month`.
- Leave clear TODO markers where more configuration will be added later.

## Files in this prompt pack

- `10-scaffold-and-layout.md` – scaffold module, directories, basic main, shared types.
- `20-pricing-embed-and-client.md` – implement `tools/generate-pricing` and the embedded pricing client.
- `30-estimation-logic-ec2-ebs.md` – implement v1 estimators for EC2 and EBS resources.
- `40-plugin-main-and-protocol.md` – implement the CLI entrypoint and JSON envelope protocol.
- `50-goreleaser-setup.md` – configure GoReleaser to build region-specific binaries with embedded JSON.
- `60-readme-and-docs.md` – update README and basic docs for usage & installation.

Apply these prompts **one at a time**, committing between major steps if you’re using git.
