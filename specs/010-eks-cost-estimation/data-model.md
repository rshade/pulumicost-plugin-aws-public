# Data Model: EKS Cluster Cost Estimation

**Date**: 2025-12-06
**Feature**: 010-eks-cost-estimation

## Overview

EKS cluster cost estimation involves a simple data model focused on cluster control plane pricing. The model captures the fixed hourly rate for EKS clusters while excluding worker node costs (handled by EC2 pricing).

## Entities

### EKS Cluster

**Purpose**: Represents an Amazon EKS cluster for cost estimation purposes.

**Attributes**:
- `hourly_rate`: Float64 - Cost per cluster-hour in USD (currently $0.10)
- `currency`: String - Currency code (always "USD" for AWS pricing)
- `region`: String - AWS region identifier (us-east-1, eu-west-1, etc.)

**Relationships**:
- None (standalone entity for pricing lookup)

**Validation Rules**:
- `hourly_rate` must be positive float
- `currency` must be "USD"
- `region` must be valid AWS region code

**State Transitions**:
- None (static pricing data)

## Data Flow

1. **Input**: ResourceDescriptor with resource_type="eks"
2. **Lookup**: Find EKS pricing for specified region
3. **Calculation**: cost_per_month = hourly_rate × 730
4. **Output**: GetProjectedCostResponse with calculated costs

## Storage

- **Format**: Embedded JSON from AWS pricing API
- **Access**: Thread-safe map lookup by region
- **Initialization**: sync.Once pattern for loading pricing data

## Schema Evolution

- **Current Version**: v1 - Basic cluster pricing
- **Future Extensions**: May add cluster size tiers if AWS introduces them
- **Backward Compatibility**: Maintain existing pricing structure

## Test Data

Sample EKS pricing data structure:
```json
{
  "service_code": "AmazonEKS",
  "product_family": "Compute",
  "attributes": {
    "usagetype": "us-east-1-AmazonEKS-Hours:perCluster",
    "operation": "CreateOperation"
  },
  "pricing": {
    "USD": "0.10"
  }
}
```</content>
<parameter name="filePath">specs/010-eks-cost-estimation/data-model.md