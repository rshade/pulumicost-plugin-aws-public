// Package plugin provides the CostSourceService gRPC implementation for AWS public pricing.
package plugin

// providerAWS is the constant identifier for the AWS provider.
const providerAWS = "aws"

// PricingNotFoundTemplate is the standard message template for missing pricing data.
// Use with fmt.Sprintf to format specific resource details.
//
// Example: fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", "t3.micro")
// Result: "EC2 instance type \"t3.micro\" not found in pricing data"
const PricingNotFoundTemplate = "%s %q not found in pricing data"

// PricingUnavailableTemplate is the standard message template for region-level pricing unavailability.
// Use with fmt.Sprintf to format service and region.
//
// Example: fmt.Sprintf(PricingUnavailableTemplate, "CloudWatch", "ap-northeast-3")
// Result: "CloudWatch pricing data not available for region ap-northeast-3"
const PricingUnavailableTemplate = "%s pricing data not available for region %s"

// Hours per month for production and development modes
const (
	HoursPerMonthProd = 730 // Production: 24 hours/day * 30 days
	HoursPerMonthDev  = 160 // Development: 8 hours/day * 5 days/week * 4 weeks
)

// Relationship types for CostAllocationLineage
const (
	RelationshipAttachedTo = "attached_to" // Direct attachment (EBS → EC2)
	RelationshipWithin     = "within"      // Containment (RDS → VPC)
	RelationshipManagedBy  = "managed_by"  // Management relationship
)
