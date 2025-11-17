# Prompt: Scaffold module and basic layout for `pulumicost-plugin-aws-public`

You are OpenCode v0.15.3 using the GrokZeroFree model.
You are operating in the repository: `pulumicost-plugin-aws-public`.

## Goal

Create the initial Go module, directory layout, and shared types for the AWS public pricing plugin.

This step is **scaffolding only**:
- No AWS pricing logic yet.
- No GoReleaser config yet (that’s a later prompt).
- Just the `go.mod`, directory structure, and core types.

## Requirements

1. **Go module initialization**

   - If `go.mod` does not exist, create it with:

     - Module path: `github.com/rshade/pulumicost-plugin-aws-public`
     - Go version: at least `1.21` (or the repo’s preferred Go version if already present).

   - If `go.mod` already exists, **do not overwrite** it; only adjust if necessary (e.g., add missing `module` line).

2. **Directory layout**

   Create the following structure if it does not exist:

   ```text
   cmd/
     pulumicost-plugin-aws-public/
       main.go
   internal/
     plugin/
       plugin.go
       types.go
     pricing/
       client.go
     config/
       config.go
   tools/
     generate-pricing/
       main.go   // stub only, real logic in later prompt
   data/
     .gitkeep   // placeholder so directory is tracked
   ```

3. **Shared types (`internal/plugin/types.go`)**

   Define the core data types used by this plugin and the JSON envelope.

   - `PluginResponse`, `PluginError`, `PluginWarning`
   - `StackEstimate`, `ResourceCostEstimate`
   - `LineItemCost`, `Assumption`

   Use the following shapes (feel free to add comments and minor cleanups):

   ```go
   package plugin

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

   type StackEstimate struct {
       Resources    []ResourceCostEstimate `json:"resources"`
       TotalMonthly float64               `json:"totalMonthly"`
       TotalHourly  float64               `json:"totalHourly"`
       Currency     string                `json:"currency"`
   }

   type ResourceCostEstimate struct {
       URN          string         `json:"urn"`
       Service      string         `json:"service"`
       ResourceType string         `json:"resourceType"`
       Region       string         `json:"region"`
       MonthlyCost  float64        `json:"monthlyCost"`
       HourlyCost   float64        `json:"hourlyCost"`
       Currency     string         `json:"currency"`
       Confidence   string         `json:"confidence"` // "high" | "medium" | "low" | "none"
       LineItems    []LineItemCost `json:"lineItems"`
       Assumptions  []Assumption   `json:"assumptions"`
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

4. **Plugin interface (`internal/plugin/plugin.go`)**

   Create a minimal plugin struct and interface for later use:

   ```go
   package plugin

   import "context"

   type StackInput struct {
       // This mirrors whatever PulumiCost core sends; for now, define a minimal structure
       // that can be extended later. At minimum include:
       Resources []ResourceInput `json:"resources"`
   }

   type ResourceInput struct {
       URN          string                 `json:"urn"`
       Provider     string                 `json:"provider"`
       Type         string                 `json:"type"`
       Name         string                 `json:"name"`
       Region       string                 `json:"region"`
       Properties   map[string]any         `json:"properties"`
   }

   type Estimator interface {
       Estimate(ctx context.Context, in *StackInput) (*StackEstimate, []PluginWarning, *PluginError)
   }

   type Plugin struct {
       Estimator Estimator
   }

   func NewPlugin(estimator Estimator) *Plugin {
       return &Plugin{Estimator: estimator}
   }
   ```

   - This is intentionally simple; later prompts will wire in a concrete estimator that uses pricing data.

5. **Stub config file (`internal/config/config.go`)**

   Create a very small config type that we can extend later:

   ```go
   package config

   type Config struct {
       Currency            string  `json:"currency"`
       AccountDiscountFactor float64 `json:"accountDiscountFactor"`
   }

   func Default() Config {
       return Config{
           Currency:             "USD",
           AccountDiscountFactor: 1.0,
       }
   }
   ```

   - Leave TODOs for loading from env/flags in later prompts.

6. **Stub pricing client (`internal/pricing/client.go`)**

   For now, just define the interface and placeholder struct; no real logic yet:

   ```go
   package pricing

   // Client provides access to embedded AWS public pricing for a specific region.
   type Client struct {
       region string
       // TODO: add parsed pricing structures and caches
   }

   const (
       // TODO: Region will be set via build-tag-specific files, e.g. embed_use1.go
       // For now, define a placeholder.
       DefaultRegion = "us-east-1"
   )

   func NewClient() (*Client, error) {
       // TODO: parse embedded JSON once and build lookup maps
       return &Client{
           region: DefaultRegion,
       }, nil
   }
   ```

   - Later prompts will add `rawPricingJSON` via `//go:embed` in region-specific files.

7. **Stub generator tool (`tools/generate-pricing/main.go`)**

   Create a small skeleton tool that accepts a `--regions` flag and **does nothing real yet**:

   ```go
   package main

   import (
       "flag"
       "fmt"
       "log"
       "strings"
   )

   func main() {
       var regionsCSV string
       flag.StringVar(&regionsCSV, "regions", "", "comma-separated list of AWS regions (e.g. us-east-1,us-west-2)")
       flag.Parse()

       if regionsCSV == "" {
           log.Fatal("missing --regions")
       }

       regions := strings.Split(regionsCSV, ",")
       fmt.Printf("Stub generate-pricing tool. Regions: %v\n", regions)

       // TODO: in a later prompt, implement:
       // - Fetch AWS pricing
       // - Trim to required services
       // - Write to data/aws_pricing_<region>.json
   }
   ```

8. **Initial main file (`cmd/pulumicost-plugin-aws-public/main.go`)**

   Create a minimal CLI entrypoint that:

   - Reads stdin (but for now doesn’t parse it).
   - Writes a dummy `PluginResponse` with `status: "error"` and `code: "NOT_IMPLEMENTED"`.

   Example:

   ```go
   package main

   import (
       "encoding/json"
       "fmt"
       "os"

       "github.com/rshade/pulumicost-plugin-aws-public/internal/plugin"
   )

   func main() {
       resp := plugin.PluginResponse{
           Version: 1,
           Status:  "error",
           Error: &plugin.PluginError{
               Code:    "NOT_IMPLEMENTED",
               Message: "pulumicost-plugin-aws-public is not implemented yet",
           },
       }

       if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
           fmt.Fprintf(os.Stderr, "failed to write response: %v\n", err)
           os.Exit(1)
       }

       os.Exit(1)
   }
   ```

## Acceptance criteria

- `go mod tidy` succeeds.
- `go build ./...` succeeds.
- The plugin binary builds and prints a `NOT_IMPLEMENTED` response when run with no input.

Make these changes now in the repo.
