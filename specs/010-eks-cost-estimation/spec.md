# Feature Specification: EKS Cluster Cost Estimation

**Feature Branch**: `010-eks-cost-estimation`  
**Created**: 2025-12-06  
**Status**: Draft  
**Input**: User description: "feat(eks): implement EKS cluster cost estimation

## Overview

Implement EKS (Elastic Kubernetes Service) cluster cost estimation. EKS has a simple, predictable pricing model with fixed hourly rates per cluster: $0.10/hour for standard support and $0.50/hour for extended support (Kubernetes versions past the 14-month standard support window).

## User Story

As a cloud cost analyst,
I want accurate EKS cluster cost estimates based on cluster count,
So that I can understand the Kubernetes management overhead in my AWS spending.

## Problem Statement

EKS is increasingly popular for container orchestration. While worker node costs are captured via EC2, the EKS control plane costs are often overlooked. Standard support costs $0.10/hour per cluster, while extended support (for Kubernetes versions past the 14-month window) costs $0.50/hour per cluster. This fixed-rate service is simple to implement and adds immediate value.

## Proposed Solution

Implement EKS pricing as a new resource type with a fixed hourly cluster rate.

### Pricing Dimensions

1. **EKS cluster hourly rate** - fixed rate per cluster-hour
   - Standard support: $0.10/hour per cluster
   - Extended support: $0.50/hour per cluster (for Kubernetes versions past the 14-month standard support window)

Note: Worker nodes (EC2 instances) are separate resources and already covered by EC2 estimation.

### Technical Approach

1. **Add Resource Type to Supports()** (`internal/plugin/supports.go`)
   - Add `\"eks\"` to supported resource types

2. **Extend Pricing Client Interface** (`internal/pricing/client.go:13-27`)
   ```go
   type PricingClient interface {
       // ... existing methods ...
       EKSClusterPricePerHour(extendedSupport bool) (float64, bool)  // Standard or Extended Support hourly rate
   }
   ```

3. **Add EKS Price Type** (`internal/pricing/types.go`)
   ```go
   type eksPrice struct {
       StandardHourlyRate float64  // $0.10/cluster-hour (standard support)
       ExtendedHourlyRate float64 // $0.50/cluster-hour (extended support)
       Currency           string
   }
   ```

4. **Update Client Struct** (`internal/pricing/client.go:29-41`)
   ```go
   type Client struct {
       // ... existing fields ...
       eksPricing *eksPrice
   }
   ```

5. **Update Pricing Initialization** (`internal/pricing/client.go:52-166`)
   - Filter by `servicecode == \"AmazonEKS\"` and `ProductFamily == \"Compute\"`
   - Extract standard cluster hourly rate:
     - `operation == \"CreateOperation\"` and `usagetype` contains `\"perCluster\"`
   - Extract extended support hourly rate:
     - `operation == \"ExtendedSupport\"` and `usagetype` contains `\"extendedSupport\"`

6. **Create Estimator Function** (`internal/plugin/projected.go`)
   ```go
   // estimateEKS calculates projected monthly cost for EKS clusters.
   // EKS has a simple fixed hourly rate per cluster (standard or extended support).
   func (p *AWSPublicPlugin) estimateEKS(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
       // Determine support type from resource.Sku or tags
       // resource.Sku = \"cluster\" (standard) or \"cluster-extended\" (extended support)
       // OR use tags: tags[\"support_type\"] == \"extended\"
       extendedSupport := resource.Sku == \"cluster-extended\" || 
                          resource.Tags[\"support_type\"] == \"extended\"
   }
   ```

7. **Add Router Case** (`internal/plugin/projected.go:70-77`)
   ```go
   case \"eks\":
       resp, err = p.estimateEKS(traceID, resource)
   ```

8. **Extend Generate-Pricing Tool** (`tools/generate-pricing/main.go`)
   - Add `AmazonEKS` service code support

### Files Likely Affected

- `internal/pricing/client.go` - Add EKS interface method, pricing struct, init logic
- `internal/pricing/types.go` - Add `eksPrice` struct
- `internal/plugin/projected.go` - Add `estimateEKS()` function, add router case
- `internal/plugin/supports.go` - Add EKS to supported resource types
- `internal/plugin/plugin_test.go` - Update mock pricing client
- `internal/plugin/projected_test.go` - Add EKS test cases
- `tools/generate-pricing/main.go` - Add EKS pricing fetch support

### ResourceDescriptor Mapping

```go
// Input - Standard Support
ResourceDescriptor{
    Provider:     \"aws\",
    ResourceType: \"eks\",
    Sku:          \"cluster\",           // Standard support
    Region:       \"us-east-1\",
    Tags:         map[string]string{},
}

// Output - Standard Support (calculation example)
// Cluster cost = 730 hrs * $0.10 = $73.00
GetProjectedCostResponse{
    UnitPrice:     0.10,               // Hourly rate
    Currency:      \"USD\",
    CostPerMonth:  73.00,
    BillingDetail: \"EKS cluster (standard support), 730 hrs/month (control plane only, excludes worker nodes)\",
}

// Input - Extended Support (via SKU)
ResourceDescriptor{
    Provider:     \"aws\",
    ResourceType: \"eks\",
    Sku:          \"cluster-extended\", // Extended support
    Region:       \"us-east-1\",
    Tags:         map[string]string{},
}

// Input - Extended Support (via tags)
ResourceDescriptor{
    Provider:     \"aws\",
    ResourceType: \"eks\",
    Sku:          \"cluster\",
    Region:       \"us-east-1\",
    Tags:         map[string]string{\"support_type\": \"extended\"},
}

// Output - Extended Support (calculation example)
// Cluster cost = 730 hrs * $0.50 = $365.00
GetProjectedCostResponse{
    UnitPrice:     0.50,               // Hourly rate
    Currency:      \"USD\",
    CostPerMonth:  365.00,
    BillingDetail: \"EKS cluster (extended support), 730 hrs/month (control plane only, excludes worker nodes)\",
}
```

## Acceptance Criteria

- [ ] `Supports()` returns `supported=true` for \"eks\" resource type
- [ ] `GetProjectedCost()` returns correct monthly cluster cost for standard support ($73.00/month)
- [ ] `GetProjectedCost()` returns correct monthly cluster cost for extended support ($365.00/month)
- [ ] Extended support can be specified via SKU (`cluster-extended`) or tags (`support_type: extended`)
- [ ] `BillingDetail` clearly states this is control plane only and indicates support type
- [ ] All 9 regional binaries include EKS pricing data (both standard and extended support)
- [ ] Thread-safe pricing lookups
- [ ] Unit tests verify calculation for both support types

## Out of Scope

- EKS worker nodes (covered by EC2 estimation)
- Fargate pod pricing
- EKS Anywhere pricing
- EKS add-ons (CNI, CoreDNS, etc.)
- Provisioned tier pricing (4XL, 8XL tiers)
- EKS Auto Mode pricing
- EKS Local Outposts pricing

## Technical Notes

### AWS Pricing API

EKS pricing is available at:
```
https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEKS/current/<region>/index.json
```

### Product Family Filters

- **EKS Cluster (Standard Support)**: `ProductFamily == \"Compute\"` with `servicecode == \"AmazonEKS\"`
  - Key attributes:
    - `usagetype`: Contains region + \"-AmazonEKS-Hours:perCluster\"
    - `operation`: \"CreateOperation\"
    - Price: $0.10/hour per cluster

- **EKS Cluster (Extended Support)**: `ProductFamily == \"Compute\"` with `servicecode == \"AmazonEKS\"`
  - Key attributes:
    - `usagetype`: Contains region + \"-AmazonEKS-Hours:extendedSupport\"
    - `operation`: \"ExtendedSupport\"
    - `tiertype`: \"HAExtended\" (optional, present in some regions)
    - Price: $0.50/hour per cluster

### Cost Calculation

```go
const hoursPerMonth = 730.0

clusterCost := hoursPerMonth * eksClusterHourlyRate
```

### EKS Pricing Constants (all regions)

```go
// EKS has uniform pricing across regions
eksStandardHourlyRate := 0.10  // $/cluster-hour (standard support)
eksExtendedHourlyRate := 0.50  // $/cluster-hour (extended support)
```

**Note**: As of 2025-12-06, the AWS Pricing API shows Extended Support at $0.50/hour ($365/month). Some documentation may reference $0.60/hour ($438/month), but the actual API pricing is $0.50/hour. The implementation should use the pricing data from the AWS Pricing API as the source of truth.

### Important Notes for BillingDetail

The response should clarify:
- This is the EKS control plane cost only
- Worker node costs are separate (EC2/Fargate)
- Each cluster incurs this fixed cost regardless of workload
- Support type (standard or extended) should be clearly indicated
- Extended support applies to Kubernetes versions past the 14-month standard support window

### Logging Requirements

```go
p.logger.Debug().
    Str(pluginsdk.FieldTraceID, traceID).
    Str(pluginsdk.FieldOperation, \"GetProjectedCost\").
    Str(\"aws_region\", p.region).
    Bool(\"extended_support\", extendedSupport).
    Float64(\"hourly_rate\", hourlyRate).
    Msg(\"EKS pricing lookup successful\")
```

## Testing Strategy

- [ ] Unit tests for `estimateEKS()` with mock pricing client (standard support)
- [ ] Unit tests for `estimateEKS()` with mock pricing client (extended support)
- [ ] Unit tests for `EKSClusterPricePerHour()` lookup (both support types)
- [ ] Test basic cluster cost calculation (standard: $73.00/month)
- [ ] Test extended support cost calculation ($365.00/month)
- [ ] Test Extended Support detection via SKU (`cluster-extended`)
- [ ] Test Extended Support detection via tags (`support_type: extended`)
- [ ] Verify BillingDetail mentions control plane only and support type
- [ ] Concurrent access test

## Related

- EC2 pattern to follow: `internal/plugin/projected.go:105-180`
- AWS EKS Pricing: https://aws.amazon.com/eks/pricing/"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Estimate EKS Cluster Costs (Priority: P1)

As a cloud cost analyst, I want to get accurate cost estimates for EKS clusters based on cluster count, so that I can understand the Kubernetes management overhead in my AWS spending.

**Why this priority**: This is the core functionality requested - providing EKS control plane cost estimates to help analysts understand their Kubernetes infrastructure costs.

**Independent Test**: Can be fully tested by calling the GetProjectedCost API with an EKS resource descriptor and verifying the returned monthly cost calculation.

**Acceptance Scenarios**:

1. **Given** a valid EKS cluster resource descriptor with resource type "eks" and SKU "cluster", **When** GetProjectedCost is called, **Then** it returns supported=true and a monthly cost of $73.00 (730 hours × $0.10/hour)
2. **Given** an EKS cluster with SKU "cluster-extended" or tag "support_type: extended", **When** GetProjectedCost is called, **Then** it returns a monthly cost of $365.00 (730 hours × $0.50/hour)
3. **Given** an EKS cluster in any supported region, **When** GetProjectedCost is called, **Then** the billing detail clearly states "control plane only, excludes worker nodes" and indicates support type
4. **Given** multiple EKS clusters (standard and extended), **When** GetProjectedCost is called for each, **Then** each returns the correct fixed hourly rate based on support type

---

### Edge Cases

- When AWS pricing API is unavailable, requests fail with error (no fallback pricing)
- System handles up to 1000 concurrent requests for EKS pricing
- What if the AWS pricing API changes the EKS service code or product family?
- No rate limiting needed as pricing data is embedded in binaries

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support "eks" as a resource type in the Supports() method
- **FR-002**: System MUST return accurate monthly EKS cluster costs based on $0.10 per cluster-hour (730 hours/month) for standard support
- **FR-003**: System MUST return accurate monthly EKS cluster costs based on $0.50 per cluster-hour (730 hours/month) for extended support
- **FR-004**: System MUST support Extended Support detection via SKU ("cluster-extended") or tags ("support_type: extended")
- **FR-005**: System MUST include clear billing details stating this covers control plane only, excluding worker nodes, and indicating support type
- **FR-006**: System MUST provide EKS pricing data (both standard and extended support) across all supported regions
- **FR-007**: System MUST ensure thread-safe access to EKS pricing lookups
- **FR-008**: System MUST log successful EKS pricing lookups with appropriate trace information including support type

### Non-Functional Requirements

- **NFR-001**: Security & Privacy - No special security measures required as EKS pricing data is public
- **NFR-002**: Compliance - No regulatory compliance requirements for EKS cost estimation

### Key Entities *(include if feature involves data)*

- **EKS Cluster (Standard Support)**: Represents an Amazon EKS cluster with a fixed hourly control plane cost of $0.10, independent of worker node configuration or usage. Applies to Kubernetes versions within the 14-month standard support window.
- **EKS Cluster (Extended Support)**: Represents an Amazon EKS cluster with a fixed hourly control plane cost of $0.50, for Kubernetes versions past the 14-month standard support window.

## Clarifications

### Session 2025-12-06
- Q: What are the expected concurrent request limits for EKS pricing lookups? → A: 1000 concurrent requests
- Q: What are the uptime/reliability expectations for EKS pricing availability? → A: 99% uptime
- Q: How should the system handle AWS pricing API failures? → A: Fail request with error
- Q: Can Extended Support pricing be detected from AWS Pricing API? → A: Yes, Extended Support is available as a separate product with `operation: "ExtendedSupport"` and `usagetype` containing `"extendedSupport"`. Price is $0.50/hour per cluster.
- Q: What security measures are required for EKS pricing data? → A: No special security needed
- Q: Are there any regulatory compliance requirements for EKS cost estimation? → A: None
- Q: What rate limiting should be applied to AWS Pricing API calls? → A: None - Pricing data embedded

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: GetProjectedCost returns correct EKS cluster cost ($73.00/month for standard, $365.00/month for extended) within 100ms for 99% of requests
- **SC-002**: System supports EKS cost estimation (both standard and extended support) across all 9 regional binaries without errors
- **SC-003**: BillingDetail accurately communicates control plane scope and support type for 100% of EKS estimates
- **SC-004**: Unit test coverage achieves 95% for EKS estimation functionality (including both support types)
- **SC-005**: Concurrent access to EKS pricing handles 1000 simultaneous requests without data corruption
- **SC-006**: EKS pricing service maintains 99% uptime availability
- **SC-007**: Extended Support detection works correctly via both SKU and tags for 100% of test cases