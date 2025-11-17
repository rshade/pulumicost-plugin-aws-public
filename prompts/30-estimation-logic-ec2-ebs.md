# Prompt: Implement EC2 & EBS estimation logic

You are OpenCode v0.15.3 using the GrokZeroFree model.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Implement the **v1 estimation logic** for:

- EC2 instances
- EBS volumes

using the embedded pricing client from `internal/pricing`.

Other services (S3, Lambda, RDS, DynamoDB) can be stubbed or partially supported, but the focus here is EC2/EBS.

## 1. Estimator struct

Under `internal/plugin`, add a new file `estimator.go` implementing a concrete `Estimator`:

```go
package plugin

import (
    "context"

    "github.com/rshade/pulumicost-plugin-aws-public/internal/config"
    "github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
)

type AWSEstimator struct {
    cfg     config.Config
    pricing *pricing.Client
}

func NewAWSEstimator(cfg config.Config, p *pricing.Client) *AWSEstimator {
    return &AWSEstimator{
        cfg:     cfg,
        pricing: p,
    }
}

func (e *AWSEstimator) Estimate(ctx context.Context, in *StackInput) (*StackEstimate, []PluginWarning, *PluginError) {
    // TODO: implement body in this prompt
}
```

## 2. EC2 estimation logic

Treat a resource as EC2 if:

- `ResourceInput.Provider == "aws"` (or starts with `"aws"`), and
- `ResourceInput.Type == "aws:ec2/instance:Instance"` (Pulumi AWS v5 type) or similar string.
  - Make this string match configurable or at least easy to change; for now just use a const.

From `ResourceInput.Properties`, read:

- `instanceType` (string)
- `region` (or fall back to `ResourceInput.Region`)

For simplicity:

- Assume:
  - `operatingSystem = "Linux"`
  - `tenancy = "Shared"`
  - `hoursPerMonth = 730` (24x7)
- Use `pricing.Client.EC2OnDemandPricePerHour(instanceType, "Linux", "Shared")`.

If price is found:

- `hourly = pricePerHour`
- `monthly = hoursPerMonth * hourly`
- `confidence = "high"`

If not found:

- `monthly = 0`, `hourly = 0`, `confidence = "none"`
- Add an assumption explaining the missing pricing entry.

Create a helper function in `estimator.go`, e.g.:

```go
func (e *AWSEstimator) estimateEC2(r ResourceInput) *ResourceCostEstimate
```

that returns `nil` if the resource is not EC2.

## 3. EBS estimation logic

Treat a resource as EBS if:

- `ResourceInput.Type == "aws:ebs/volume:Volume"` (again, use a const).

From `ResourceInput.Properties`:

- `volumeType` (string, e.g. `gp2`, `gp3`)
- `size` (GB, integer or float; if absent, assume 8 GB and mark with an assumption)

Lookup:

- `pricePerGBMonth := pricingClient.EBSPricePerGBMonth(volumeType)`

Compute:

- `monthly = sizeGB * pricePerGBMonth`
- `hourly = monthly / 730` (rough approximation)

Confidence:

- `"high"` if both `volumeType` and `size` exist and price is found.
- `"medium"` if `size` was assumed.
- `"none"` if pricing not found.

Similarly, implement a helper:

```go
func (e *AWSEstimator) estimateEBS(r ResourceInput) *ResourceCostEstimate
```

that returns `nil` if the resource is not EBS.

## 4. Orchestrating estimate across resources

Implement the body of `Estimate`:

```go
func (e *AWSEstimator) Estimate(ctx context.Context, in *StackInput) (*StackEstimate, []PluginWarning, *PluginError) {
    if in == nil {
        return nil, nil, &PluginError{
            Code:    "INVALID_INPUT",
            Message: "nil input",
        }
    }

    // Sanity: ensure pricing region matches resources, but do not error yet.
    pluginRegion := e.pricing.Region()
    var resourceRegions = make(map[string]struct{})
    for _, r := range in.Resources {
        if r.Region != "" {
            resourceRegions[r.Region] = struct{}{}
        }
    }

    // If all resources are in one region and it differs from pluginRegion,
    // we will return an UNSUPPORTED_REGION error.
    if len(resourceRegions) == 1 {
        for region := range resourceRegions {
            if region != "" && region != pluginRegion {
                return nil, nil, &PluginError{
                    Code:    "UNSUPPORTED_REGION",
                    Message: "stack resources are in a different region than this plugin binary",
                    Meta: map[string]any{
                        "pluginRegion":   pluginRegion,
                        "requiredRegion": region,
                    },
                }
            }
        }
    }

    // Otherwise, proceed and estimate only for resources that match pluginRegion or have empty region.
    var (
        resources []ResourceCostEstimate
        warnings  []PluginWarning
        totalMonthly float64
    )

    for _, r := range in.Resources {
        if r.Region != "" && r.Region != pluginRegion {
            // Different region – skip for now, but add a warning once.
            warnings = append(warnings, PluginWarning{
                Code:    "PARTIAL_UNSUPPORTED_REGION",
                Message: "some resources are in regions not supported by this plugin binary",
                Meta: map[string]any{
                    "pluginRegion": pluginRegion,
                    "resourceRegion": r.Region,
                },
            })
            continue
        }

        if est := e.estimateEC2(r); est != nil {
            resources = append(resources, *est)
            totalMonthly += est.MonthlyCost
            continue
        }

        if est := e.estimateEBS(r); est != nil {
            resources = append(resources, *est)
            totalMonthly += est.MonthlyCost
            continue
        }

        // For non-EC2/EBS resources, we currently do nothing.
        // Future prompts can add more service handlers here.
    }

    hourly := totalMonthly / 730.0
    stack := &StackEstimate{
        Resources:    resources,
        TotalMonthly: totalMonthly,
        TotalHourly:  hourly,
        Currency:     e.pricing.Currency(),
    }

    return stack, warnings, nil
}
```

Feel free to refine this logic as long as the behavior is equivalent.

## 5. Assumptions and line items

When building `ResourceCostEstimate`, make sure to populate:

- `Service`: `"ec2"` or `"ebs"`
- `ResourceType`: the Pulumi `Type` string.
- `Currency`: from `pricing.Client`.
- `LineItems`: at least one line item for the main charge:
  - For EC2:
    - `Description`: e.g. `"On-demand Linux/Shared instance"`
    - `Unit`: `"Hrs"`
    - `Quantity`: `730`
    - `Rate`: `pricePerHour`
    - `Cost`: `monthly`
  - For EBS:
    - `Description`: e.g. `"EBS gp2 storage"`
    - `Unit`: `"GB-Mo"`
    - `Quantity`: `sizeGB`
    - `Rate`: `pricePerGBMonth`
    - `Cost`: `monthly`

- `Assumptions`: when you fall back to defaults (e.g., missing size, or default hours per month).

## 6. Tests

Add tests for the estimator:

- `internal/plugin/estimator_test.go`:
  - Use a fake `pricing.Client` (you can define a small test-only type in the test file that satisfies the subset of methods you use).
  - Or temporarily export enough fields/methods from `pricing.Client` to construct a minimal test instance with deterministic prices.
  - Test scenarios:
    - EC2 + EBS resources with known prices.
    - Resource with different region → `UNSUPPORTED_REGION` when all resources share that region.
    - Mixed regions → partial warnings and estimates, not a hard error.

## Acceptance criteria

- `go test ./...` passes with estimator tests included.
- `go build ./...` succeeds.
- When run with a simple input containing EC2/EBS resources matching the plugin region, the plugin produces non-zero cost estimates (after we wire up main in a later prompt).

Implement these changes now.
