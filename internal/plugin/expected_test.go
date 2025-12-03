package plugin

import (
	"math"
	"testing"
)

func TestGetExpectedRange(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		sku          string
		region       string
		wantFound    bool
		wantMonthly  float64
		wantTolerance float64
	}{
		{
			name:          "EC2 t3.micro us-east-1",
			resourceType:  "ec2",
			sku:           "t3.micro",
			region:        "us-east-1",
			wantFound:     true,
			wantMonthly:   7.592,
			wantTolerance: 1.0,
		},
		{
			name:          "EBS gp2 us-east-1",
			resourceType:  "ebs",
			sku:           "gp2",
			region:        "us-east-1",
			wantFound:     true,
			wantMonthly:   0.80,
			wantTolerance: 5.0,
		},
		{
			name:         "unsupported resource type",
			resourceType: "s3",
			sku:          "standard",
			region:       "us-east-1",
			wantFound:    false,
		},
		{
			name:         "unsupported region",
			resourceType: "ec2",
			sku:          "t3.micro",
			region:       "eu-west-1",
			wantFound:    false,
		},
		{
			name:         "unsupported instance type",
			resourceType: "ec2",
			sku:          "m5.large",
			region:       "us-east-1",
			wantFound:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := GetExpectedRange(tt.resourceType, tt.sku, tt.region)

			if found != tt.wantFound {
				t.Errorf("GetExpectedRange() found = %v, want %v", found, tt.wantFound)
			}

			if tt.wantFound {
				if got.MonthlyEstimate != tt.wantMonthly {
					t.Errorf("GetExpectedRange() monthly = %v, want %v", got.MonthlyEstimate, tt.wantMonthly)
				}
				if got.TolerancePercent != tt.wantTolerance {
					t.Errorf("GetExpectedRange() tolerance = %v, want %v", got.TolerancePercent, tt.wantTolerance)
				}
			}
		})
	}
}

func TestExpectedCostRange_MinMax(t *testing.T) {
	tests := []struct {
		name     string
		range_   ExpectedCostRange
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "EC2 1% tolerance",
			range_: ExpectedCostRange{
				MonthlyEstimate:  7.592,
				TolerancePercent: 1.0,
			},
			wantMin: 7.51608, // 7.592 * 0.99
			wantMax: 7.66792, // 7.592 * 1.01
		},
		{
			name: "EBS 5% tolerance",
			range_: ExpectedCostRange{
				MonthlyEstimate:  0.80,
				TolerancePercent: 5.0,
			},
			wantMin: 0.76, // 0.80 * 0.95
			wantMax: 0.84, // 0.80 * 1.05
		},
		{
			name: "zero tolerance",
			range_: ExpectedCostRange{
				MonthlyEstimate:  10.0,
				TolerancePercent: 0,
			},
			wantMin: 10.0,
			wantMax: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin := tt.range_.Min()
			gotMax := tt.range_.Max()

			if math.Abs(gotMin-tt.wantMin) > 0.00001 {
				t.Errorf("Min() = %v, want %v", gotMin, tt.wantMin)
			}
			if math.Abs(gotMax-tt.wantMax) > 0.00001 {
				t.Errorf("Max() = %v, want %v", gotMax, tt.wantMax)
			}
		})
	}
}

func TestIsWithinTolerance(t *testing.T) {
	tests := []struct {
		name       string
		actual     float64
		expected   float64
		tolerance  float64
		want       bool
	}{
		{
			name:      "exactly at expected",
			actual:    7.592,
			expected:  7.592,
			tolerance: 1.0,
			want:      true,
		},
		{
			name:      "at lower boundary (1%)",
			actual:    7.51608,
			expected:  7.592,
			tolerance: 1.0,
			want:      true,
		},
		{
			name:      "at upper boundary (1%)",
			actual:    7.66792,
			expected:  7.592,
			tolerance: 1.0,
			want:      true,
		},
		{
			name:      "just below tolerance",
			actual:    7.51,
			expected:  7.592,
			tolerance: 1.0,
			want:      false,
		},
		{
			name:      "just above tolerance",
			actual:    7.67,
			expected:  7.592,
			tolerance: 1.0,
			want:      false,
		},
		{
			name:      "5% tolerance within range",
			actual:    0.82,
			expected:  0.80,
			tolerance: 5.0,
			want:      true,
		},
		{
			name:      "5% tolerance outside range",
			actual:    0.90,
			expected:  0.80,
			tolerance: 5.0,
			want:      false,
		},
		{
			name:      "zero expected with zero actual",
			actual:    0,
			expected:  0,
			tolerance: 1.0,
			want:      true,
		},
		{
			name:      "zero expected with non-zero actual",
			actual:    0.01,
			expected:  0,
			tolerance: 1.0,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWithinTolerance(tt.actual, tt.expected, tt.tolerance)
			if got != tt.want {
				t.Errorf("IsWithinTolerance(%v, %v, %v) = %v, want %v",
					tt.actual, tt.expected, tt.tolerance, got, tt.want)
			}
		})
	}
}

func TestCalculateExpectedActualCost(t *testing.T) {
	tests := []struct {
		name          string
		projected     float64
		runtimeHours  float64
		want          float64
	}{
		{
			name:         "30 minutes for t3.micro",
			projected:    7.592,
			runtimeHours: 0.5,
			want:         0.0052, // 7.592 * (0.5 / 730) ≈ 0.0052
		},
		{
			name:         "1 hour for t3.micro",
			projected:    7.592,
			runtimeHours: 1.0,
			want:         0.0104, // 7.592 * (1 / 730) ≈ 0.0104
		},
		{
			name:         "full month",
			projected:    7.592,
			runtimeHours: 730,
			want:         7.592,
		},
		{
			name:         "zero runtime",
			projected:    7.592,
			runtimeHours: 0,
			want:         0,
		},
		{
			name:         "30 minutes for EBS",
			projected:    0.80,
			runtimeHours: 0.5,
			want:         0.00054795, // 0.80 * (0.5 / 730)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateExpectedActualCost(tt.projected, tt.runtimeHours)
			// Allow small floating point tolerance
			if math.Abs(got-tt.want) > 0.0001 {
				t.Errorf("CalculateExpectedActualCost(%v, %v) = %v, want %v",
					tt.projected, tt.runtimeHours, got, tt.want)
			}
		})
	}
}

func TestExpectedCostRangesContainReferenceDate(t *testing.T) {
	for key, r := range ExpectedCostRanges {
		t.Run(key, func(t *testing.T) {
			if r.ReferenceDate == "" {
				t.Errorf("ExpectedCostRanges[%q] missing ReferenceDate", key)
			}
			// Verify date format is ISO-like (YYYY-MM-DD)
			if len(r.ReferenceDate) != 10 || r.ReferenceDate[4] != '-' || r.ReferenceDate[7] != '-' {
				t.Errorf("ExpectedCostRanges[%q].ReferenceDate = %q, want YYYY-MM-DD format",
					key, r.ReferenceDate)
			}
		})
	}
}
