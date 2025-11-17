# Prompt: Implement pricing generator and embedded pricing client

You are OpenCode v0.15.3 using the GrokZeroFree model.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Implement:

1. A build-time tool (`tools/generate-pricing`) that:
   - Accepts a list of regions.
   - Produces trimmed pricing JSON files under `data/`.
   - Is easy to wire into GoReleaser `before.hooks`.

2. An embedded pricing client under `internal/pricing` that:
   - Uses `//go:embed` to load `data/aws_pricing_<region>.json` into the binary.
   - Parses it into lookup maps for EC2 and EBS pricing.
   - Exposes simple getter methods for EC2 and EBS prices.

This prompt does **not** yet cover S3/Lambda/RDS/DynamoDB; it’s OK to leave TODO stubs for those.

## 1. Pricing file format

Design a **small internal JSON schema** for per-region pricing files, independent of AWS’s giant raw format. For example:

```jsonc
{
  "region": "us-east-1",
  "currency": "USD",
  "ec2": {
    "onDemand": [
      {
        "instanceType": "t3.micro",
        "operatingSystem": "Linux",
        "tenancy": "Shared",
        "pricePerHour": 0.0104
      }
    ]
  },
  "ebs": {
    "gp2": {
      "pricePerGBMonth": 0.10
    },
    "gp3": {
      "pricePerGBMonth": 0.08
    }
  }
}
```

You may adjust the schema as long as it stays simple and efficient to query.

## 2. `tools/generate-pricing` implementation

Update `tools/generate-pricing/main.go` to:

- Accept:
  - `--regions` (required, comma-separated).
  - `--out-dir` (optional, default `./data`).
- For each region:
  - Call a helper function to fetch and trim AWS pricing.
  - Write `aws_pricing_<region>.json` into the output directory.

Because this environment may not have AWS credentials or network access, organize the code so that:

- The actual HTTP calls to AWS are isolated in a function like:

  ```go
  func FetchRawAWSPricing(region string) ([]byte, error)
  ```

- The trimming logic operates on a **simplified in-memory representation**, so it’s easy to test independently and later plug in real data.

For now, you can:

- Implement basic structure and parsing assuming AWS’s price list JSON format.
- Add TODO comments where you would actually parse/filter real AWS JSON.
- Optionally, add a `--dummy` flag that writes **hard-coded sample** pricing data (e.g., only `t3.micro` and `gp2`) for development/testing.

Example:

```go
// If --dummy is set, bypass network and write a small hard-coded pricing file for each region.
```

This will let the plugin be functional in tests without external dependencies.

## 3. Embedded pricing via `//go:embed`

Under `internal/pricing`, create region-specific files with build tags:

- `internal/pricing/embed_use1.go` for `us-east-1`
- `internal/pricing/embed_usw2.go` for `us-west-2`
- `internal/pricing/embed_euw1.go` for `eu-west-1`

Example for `us-east-1`:

```go
// internal/pricing/embed_use1.go
//go:build region_use1

package pricing

import _ "embed"

//go:embed ../../data/aws_pricing_us-east-1.json
var rawPricingJSON []byte

const Region = "us-east-1"
```

For now, it’s OK if the referenced `data/aws_pricing_*.json` files do not exist in git; they will be generated at release time.

Also create a fallback file with a default region and an empty `rawPricingJSON` to keep development `go build` happy when no build tags are set:

```go
// internal/pricing/embed_default.go
//go:build !region_use1 && !region_usw2 && !region_euw1

package pricing

var rawPricingJSON []byte
const Region = "us-east-1"
```

## 4. Implement `Client` parsing and lookups

Update `internal/pricing/client.go` to:

1. Define an internal struct that matches your trimmed JSON format, e.g.:

   ```go
   type pricingFile struct {
       Region   string          `json:"region"`
       Currency string          `json:"currency"`
       EC2      ec2PricingBlock `json:"ec2"`
       EBS      ebsPricingBlock `json:"ebs"`
   }

   type ec2PricingBlock struct {
       OnDemand []ec2OnDemandPrice `json:"onDemand"`
   }

   type ec2OnDemandPrice struct {
       InstanceType    string  `json:"instanceType"`
       OperatingSystem string  `json:"operatingSystem"`
       Tenancy         string  `json:"tenancy"`
       PricePerHour    float64 `json:"pricePerHour"`
   }

   type ebsPricingBlock struct {
       Volumes map[string]ebsVolumePrice `json:"volumes"`
   }

   type ebsVolumePrice struct {
       PricePerGBMonth float64 `json:"pricePerGBMonth"`
   }
   ```

   You may adjust field names and nesting if you prefer.

2. Parse `rawPricingJSON` once using `sync.Once`:

   ```go
   type Client struct {
       region   string
       currency string

       once sync.Once
       err  error

       ec2Index map[string]ec2OnDemandPrice // key: instanceType + "/" + operatingSystem + "/" + tenancy
       ebsIndex map[string]ebsVolumePrice   // key: volumeType
   }

   func NewClient() (*Client, error) {
       c := &Client{
           region: Region,
       }
       if len(rawPricingJSON) == 0 {
           // allow a zero-pricing client but mark error for real use
           return c, nil
       }
       if err := c.init(); err != nil {
           return nil, err
       }
       return c, nil
   }

   func (c *Client) init() error {
       var pf pricingFile
       if err := json.Unmarshal(rawPricingJSON, &pf); err != nil {
           return err
       }
       c.currency = pf.Currency
       // build indexes...
       return nil
   }
   ```

3. Expose lookup methods:

   ```go
   func (c *Client) Region() string  { return c.region }
   func (c *Client) Currency() string { return c.currency }

   func (c *Client) EC2OnDemandPricePerHour(instanceType, operatingSystem, tenancy string) (float64, bool) {
       // look up in c.ec2Index, return (price, true) or (0, false)
   }

   func (c *Client) EBSPricePerGBMonth(volumeType string) (float64, bool) {
       // look up in c.ebsIndex
   }
   ```

   - Keep the API minimal and focused on v1 services (EC2 + EBS).

## 5. Testing

- Add a small unit test file `internal/pricing/client_test.go` that:
  - Creates a `Client` backed by a small hard-coded `rawPricingJSON` (you can override the package-level variable in test).
  - Verifies that:
    - `NewClient()` parses correctly.
    - EC2 and EBS lookup methods return the expected values.

## Acceptance criteria

- `go test ./...` passes (with at least a basic test for `internal/pricing`).
- `go build ./...` still succeeds with no build tags set.
- `tools/generate-pricing` builds and supports:
  - `--regions`
  - `--out-dir`
  - optional `--dummy` mode for local development.

Implement these changes now.
