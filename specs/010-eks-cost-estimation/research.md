# Research: EKS Cluster Cost Estimation

**Date**: 2025-12-06
**Feature**: 010-eks-cost-estimation

## Research Questions & Findings

### Q1: AWS EKS Pricing API Structure
**Decision**: Use existing AWS pricing API JSON structure with servicecode="AmazonEKS"
**Rationale**: Follows established pattern from EC2/S3 pricing in the codebase
**Alternatives considered**: Custom pricing endpoint (rejected - increases complexity and maintenance)

**Findings**:
- EKS pricing available at: `https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEKS/current/<region>/index.json`
- Service code: `AmazonEKS`
- Product family filter: `ProductFamily == "Compute"`
- Usage type pattern: Contains region + "-AmazonEKS-Hours:perCluster"
- Operation: "CreateOperation"

### Q2: EKS Pricing Data Extraction
**Decision**: Extract hourly cluster rate from pricing JSON using existing patterns
**Rationale**: Consistent with how EC2/S3 pricing is extracted in tools/generate-pricing
**Alternatives considered**: Hardcoded rates (rejected - pricing can change, violates embedded data principle)

**Findings**:
- EKS has uniform pricing across all regions ($0.10/hour per cluster)
- Single pricing dimension (cluster-hours)
- No additional attributes needed beyond service filters
- Pricing structure matches existing AWS pricing API format

### Q3: Generate-Pricing Tool Integration
**Decision**: Extend tools/generate-pricing/main.go to support AmazonEKS service
**Rationale**: Follows existing pattern for EC2, S3, RDS pricing generation
**Alternatives considered**: Separate tool (rejected - violates DRY, increases maintenance)

**Findings**:
- Tool already supports multiple AWS services via service code filtering
- Need to add "AmazonEKS" to supported services list
- Output follows existing embed_*.go pattern
- Region-specific builds will include EKS pricing data

### Q4: EKS Pricing Variations
**Decision**: Assume uniform pricing but validate per region
**Rationale**: AWS documentation states uniform pricing, but implementation should handle potential variations
**Alternatives considered**: Single global rate (rejected - future-proofs for potential regional differences)

**Findings**:
- Current AWS pricing: $0.10/hour per cluster across all regions
- No tiered pricing or volume discounts for EKS control plane
- Worker node costs handled separately via EC2 pricing
- Implementation should extract actual rates from pricing API data

### Q5: Cost Calculation Formula
**Decision**: cost_per_month = hourly_rate × 730 (hours in month)
**Rationale**: Consistent with existing AWS service calculations in the codebase
**Alternatives considered**: Custom month calculation (rejected - violates consistency)

**Findings**:
- Standard 730 hours per month (365 × 24 ÷ 12)
- Used across all existing pricing calculations
- Matches AWS billing cycle assumptions

### Q6: Thread Safety for EKS Pricing
**Decision**: Use sync.Once for initialization, RWMutex for concurrent access
**Rationale**: Follows existing pricing client patterns for thread safety
**Alternatives considered**: No synchronization (rejected - violates constitution thread safety requirement)

**Findings**:
- Existing pricing clients use sync.Once + maps for thread-safe access
- EKS pricing should follow same pattern
- Concurrent gRPC calls require thread-safe pricing lookups

## Implementation Approach

Based on research, EKS implementation will:

1. **Extend Pricing Client**: Add EKSClusterPricePerHour() method following existing interface patterns
2. **Add Pricing Types**: Create eksPrice struct in internal/pricing/types.go
3. **Update Initialization**: Filter AmazonEKS service data in client initialization
4. **Create Estimator**: Add estimateEKS() function in internal/plugin/projected.go
5. **Update Router**: Add "eks" case to Supports() and GetProjectedCost() routing
6. **Generate Pricing**: Extend tools/generate-pricing to include AmazonEKS data
7. **Add Tests**: Unit tests for pricing lookup and cost calculation

All changes follow established codebase patterns for consistency and maintainability.</content>
<parameter name="filePath">specs/010-eks-cost-estimation/research.md