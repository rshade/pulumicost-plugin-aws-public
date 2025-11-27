package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AWSPublicPlugin implements the pluginsdk.Plugin interface for AWS public pricing.
type AWSPublicPlugin struct {
	region  string
	pricing pricing.PricingClient
	logger  zerolog.Logger
}

// NewAWSPublicPlugin creates a new AWSPublicPlugin instance.
// The region should match the region for which pricing data is embedded.
// The logger should be created using pluginsdk.NewPluginLogger for consistency.
func NewAWSPublicPlugin(region string, pricingClient pricing.PricingClient, logger zerolog.Logger) *AWSPublicPlugin {
	return &AWSPublicPlugin{
		region:  region,
		pricing: pricingClient,
		logger:  logger,
	}
}

// getTraceID extracts the trace_id from context or generates a UUID if not present.
// This implements the workaround for missing interceptor support in ServeConfig.
// See research.md U1 Remediation for details.
func (p *AWSPublicPlugin) getTraceID(ctx context.Context) string {
	// First try SDK helper (works if interceptor was somehow registered)
	traceID := pluginsdk.TraceIDFromContext(ctx)
	if traceID != "" {
		return traceID
	}

	// Fallback: read directly from gRPC metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(pluginsdk.TraceIDMetadataKey); len(values) > 0 {
			return values[0]
		}
	}

	// Generate UUID if not present (per FR-003)
	return uuid.New().String()
}

// logErrorWithID logs an error using a pre-captured trace ID.
// Use this when you've already extracted the trace ID to ensure consistency
// between error objects and log entries.
func (p *AWSPublicPlugin) logErrorWithID(traceID, operation string, err error, code pbc.ErrorCode) {
	p.logger.Error().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, operation).
		Str(pluginsdk.FieldErrorCode, code.String()).
		Err(err).
		Msg("request failed")
}

// newErrorWithID creates a gRPC error with trace_id in the error details using a pre-captured trace ID.
// Use this when you've already extracted the trace ID to ensure consistency
// between error objects and log entries.
func (p *AWSPublicPlugin) newErrorWithID(traceID string, grpcCode codes.Code, msg string, errorCode pbc.ErrorCode) error {
	errDetail := &pbc.ErrorDetail{
		Code:    errorCode,
		Message: msg,
		Details: map[string]string{
			"trace_id": traceID,
		},
	}

	st := status.New(grpcCode, msg)
	st, _ = st.WithDetails(errDetail)
	return st.Err()
}

// Name returns the plugin name identifier.
func (p *AWSPublicPlugin) Name() string {
	return "pulumicost-plugin-aws-public"
}

// GetActualCost retrieves actual cost for a resource based on runtime.
// Uses fallback formula: actual_cost = projected_monthly_cost × (runtime_hours / 730)
//
// The proto API uses ResourceId (string) which we expect to be a JSON-encoded
// ResourceDescriptor. If ResourceId is empty, we fall back to extracting
// resource info from the Tags map.
func (p *AWSPublicPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	// Validate request
	if req == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "missing request", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetActualCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Validate timestamps (proto uses Start/End)
	if req.Start == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "missing Start timestamp", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetActualCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}
	if req.End == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "missing End timestamp", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetActualCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Parse timestamps
	fromTime := req.Start.AsTime()
	toTime := req.End.AsTime()

	// Calculate runtime hours
	runtimeHours, err := calculateRuntimeHours(fromTime, toTime)
	if err != nil {
		statusErr := p.newErrorWithID(traceID, codes.InvalidArgument, fmt.Sprintf("invalid time range: %v", err), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetActualCost", statusErr, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, statusErr
	}

	// Parse ResourceDescriptor from ResourceId (JSON) or Tags
	resource, err := p.parseResourceFromRequest(req)
	if err != nil {
		p.logErrorWithID(traceID, "GetActualCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Handle zero duration - return $0 with single result
	if runtimeHours == 0 {
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "GetActualCost").
			Float64("cost_monthly", 0).
			Float64("usage_amount", runtimeHours).
			Str("usage_unit", "hours").
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("cost calculated")

		return &pbc.GetActualCostResponse{
			Results: []*pbc.ActualCostResult{{
				Timestamp: req.Start,
				Cost:      0,
				Source:    "aws-public-fallback",
			}},
		}, nil
	}

	// Get projected monthly cost using helper
	projectedResp, err := p.getProjectedForResource(traceID, resource)
	if err != nil {
		// Note: Use UNSPECIFIED as the error could be various types from projected cost calculation
		p.logErrorWithID(traceID, "GetActualCost", err, pbc.ErrorCode_ERROR_CODE_UNSPECIFIED)
		return nil, err
	}

	// Apply formula: actual_cost = projected_monthly_cost × (runtime_hours / 730)
	actualCost := projectedResp.CostPerMonth * (runtimeHours / hoursPerMonth)

	p.logger.Info().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetActualCost").
		Str(pluginsdk.FieldResourceType, resource.ResourceType).
		Str("aws_service", resource.ResourceType).
		Str("aws_region", resource.Region).
		Float64("cost_monthly", actualCost).
		Float64("usage_amount", runtimeHours).
		Str("usage_unit", "hours").
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("cost calculated")

	return &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{{
			Timestamp:   req.Start,
			Cost:        actualCost,
			UsageAmount: runtimeHours,
			UsageUnit:   "hours",
			Source:      formatActualBillingDetail(projectedResp.BillingDetail, runtimeHours, actualCost),
		}},
	}, nil
}

// parseResourceFromRequest extracts a ResourceDescriptor from the request.
// It first tries to parse ResourceId as JSON, then falls back to Tags.
func (p *AWSPublicPlugin) parseResourceFromRequest(req *pbc.GetActualCostRequest) (*pbc.ResourceDescriptor, error) {
	// Try parsing ResourceId as JSON-encoded ResourceDescriptor
	if req.ResourceId != "" {
		var resource pbc.ResourceDescriptor
		if err := json.Unmarshal([]byte(req.ResourceId), &resource); err == nil {
			return &resource, nil
		}
		// If JSON parsing fails, treat ResourceId as a simple ID and use Tags
	}

	// Fall back to extracting from Tags
	tags := req.Tags
	if tags == nil {
		return nil, status.Error(codes.InvalidArgument, "missing resource information: provide ResourceId as JSON or use Tags")
	}

	// Extract resource info from tags
	resource := &pbc.ResourceDescriptor{
		Provider:     tags["provider"],
		ResourceType: tags["resource_type"],
		Sku:          tags["sku"],
		Region:       tags["region"],
		Tags:         make(map[string]string),
	}

	// Copy remaining tags (excluding the resource descriptor fields)
	for k, v := range tags {
		switch k {
		case "provider", "resource_type", "sku", "region":
			// Skip - already extracted
		default:
			resource.Tags[k] = v
		}
	}

	// Validate required fields
	if resource.Provider == "" || resource.ResourceType == "" || resource.Sku == "" || resource.Region == "" {
		return nil, status.Error(codes.InvalidArgument, "resource information incomplete: need provider, resource_type, sku, region in ResourceId or Tags")
	}

	return resource, nil
}
