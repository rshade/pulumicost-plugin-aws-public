//go:build integration

package plugin_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"testing"
	"time"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestIntegration_APSoutheast1_Binary tests the ap-southeast-1 binary end-to-end (T014)
// Run with: go test -tags=integration ./internal/plugin/...
func TestIntegration_APSoutheast1_Binary(t *testing.T) {
	// Build the binary with region_apse1 tag
	t.Log("Building ap-southeast-1 binary...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_apse1",
		"-o", "../../dist/test-pulumicost-plugin-aws-public-ap-southeast-1",
		"../../cmd/pulumicost-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("../../dist/test-pulumicost-plugin-aws-public-ap-southeast-1")

	// Start the binary
	t.Log("Starting ap-southeast-1 binary...")
	cmd := exec.Command("../../dist/test-pulumicost-plugin-aws-public-ap-southeast-1")
	cmd.Dir, _ = os.Getwd()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start binary: %v", err)
	}
	defer cmd.Process.Kill()

	// Read PORT announcement from stdout
	var port int
	scanner := bufio.NewScanner(stdout)
	portRegex := regexp.MustCompile(`^PORT=(\d+)$`)

	timeout := time.After(5 * time.Second)
	portChan := make(chan int, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if matches := portRegex.FindStringSubmatch(line); len(matches) == 2 {
				if p, err := strconv.Atoi(matches[1]); err == nil {
					portChan <- p
					return
				}
			}
		}
	}()

	select {
	case port = <-portChan:
		t.Logf("Binary announced PORT=%d", port)
	case <-timeout:
		t.Fatal("Timeout waiting for PORT announcement")
	}

	// Give the server a moment to start listening
	time.Sleep(500 * time.Millisecond)

	// Connect to the gRPC server
	t.Log("Connecting to gRPC server...")
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: Name()
	t.Run("Name", func(t *testing.T) {
		resp, err := client.Name(ctx, &pbc.NameRequest{})
		if err != nil {
			t.Fatalf("Name() failed: %v", err)
		}
		t.Logf("Plugin name: %s", resp.Name)
		if resp.Name != "pulumicost-plugin-aws-public" {
			t.Errorf("Expected name 'pulumicost-plugin-aws-public', got %q", resp.Name)
		}
	})

	// TODO(rshade): Re-enable Supports() tests once gRPC handler is implemented
	// Supports() is not exposed via gRPC by pluginsdk - it works in unit tests but not integration tests.
	// Tracking issues:
	//   - pulumicost-spec#64: Add Supports() RPC method to proto
	//   - pulumicost-core#160: Implement Supports() gRPC handler in pluginsdk
	// For now, GetProjectedCost() properly validates regions and rejects mismatches.

	// Test 4: GetProjectedCost() - t3.micro in ap-southeast-1
	t.Run("GetProjectedCost_t3micro_Singapore", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "ap-southeast-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("t3.micro in ap-southeast-1: $%.2f/month (hourly: $%.4f, currency: %s)",
			resp.CostPerMonth, resp.UnitPrice, resp.Currency)
		t.Logf("Billing detail: %s", resp.BillingDetail)

		if resp.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", resp.Currency)
		}
		if resp.UnitPrice <= 0 {
			t.Errorf("Expected positive unit price, got %.4f", resp.UnitPrice)
		}
		if resp.CostPerMonth <= 0 {
			t.Errorf("Expected positive monthly cost, got %.2f", resp.CostPerMonth)
		}
		// Verify it's roughly hourly_rate * 730
		expectedCost := resp.UnitPrice * 730.0
		if resp.CostPerMonth < expectedCost*0.99 || resp.CostPerMonth > expectedCost*1.01 {
			t.Errorf("Monthly cost %.2f doesn't match expected %.2f (hourly * 730)",
				resp.CostPerMonth, expectedCost)
		}
	})

	// Test 5: GetProjectedCost() - EBS gp3 in ap-southeast-1
	t.Run("GetProjectedCost_gp3_Singapore", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ebs",
				Sku:          "gp3",
				Region:       "ap-southeast-1",
				Tags: map[string]string{
					"size": "100",
				},
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("gp3 100GB in ap-southeast-1: $%.2f/month (per-GB: $%.4f)",
			resp.CostPerMonth, resp.UnitPrice)
		t.Logf("Billing detail: %s", resp.BillingDetail)

		if resp.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", resp.Currency)
		}
		if resp.UnitPrice <= 0 {
			t.Errorf("Expected positive per-GB price, got %.4f", resp.UnitPrice)
		}
		// Verify it's roughly per_gb_rate * 100
		expectedCost := resp.UnitPrice * 100.0
		if resp.CostPerMonth < expectedCost*0.99 || resp.CostPerMonth > expectedCost*1.01 {
			t.Errorf("Monthly cost %.2f doesn't match expected %.2f (per-GB * 100)",
				resp.CostPerMonth, expectedCost)
		}
	})

	// Test 6: GetProjectedCost() - Wrong region (should fail)
	t.Run("GetProjectedCost_WrongRegion", func(t *testing.T) {
		_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		})
		if err == nil {
			t.Error("Expected error for wrong region, got nil")
		} else {
			t.Logf("Correctly rejected wrong region with error: %v", err)
		}
	})

	t.Log("Integration test completed successfully!")
}
