//go:build integration

package integration

import (
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const imageName = "finfocus-aws:test"

// TestKubernetesDeploymentWithKind verifies the plugin deploys and runs correctly on a local Kubernetes cluster.
//
// This test validates end-to-end Kubernetes deployment, including cluster creation,
// image loading, pod readiness, port forwarding, and health endpoint verification.
// It ensures the containerized plugin integrates correctly with Kubernetes orchestration.
//
// Test workflow:
//  1. Check if kind and kubectl CLIs are available (skip if not)
//  2. Create a Kind cluster (test-aws-cluster)
//  3. Load the finfocus-aws Docker image into the cluster
//  4. Deploy the application using kubectl apply on test/k8s/deployment.yaml
//  5. Wait for pod to be Running and Ready (2 minute timeout)
//  6. Set up port forwarding (8001:8001, 9090:9090)
//  7. Verify health endpoint responds via port-forward
//  8. Cleanup: delete the Kind cluster
//
// Prerequisites:
//   - kind CLI available and accessible in PATH
//   - kubectl CLI available and accessible in PATH
//   - Docker image finfocus-aws:test already built and available locally
//
// Run with: go test -tags=integration -run TestKubernetesDeploymentWithKind ./test/integration/...
func TestKubernetesDeploymentWithKind(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if kind is available
	if _, err := exec.LookPath("kind"); err != nil {
		t.Skip("kind not available, skipping test")
	}

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not available, skipping test")
	}

	clusterName := "test-aws-cluster"

	// Create Kind cluster
	t.Log("Creating Kind cluster...")
	cmd := exec.Command("kind", "create", "cluster", "--name", clusterName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create Kind cluster: %v\nOutput: %s", err, output)
	}
	defer func() {
		if err := exec.Command("kind", "delete", "cluster", "--name", clusterName).Run(); err != nil {
			t.Logf("Failed to delete Kind cluster %s: %v", clusterName, err)
		}
	}()

	// Load Docker image into Kind
	t.Log("Loading image into Kind...")
	cmd = exec.Command("kind", "load", "docker-image", imageName, "--name", clusterName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to load image into Kind: %v\nOutput: %s", err, output)
	}

	// Deploy the application
	t.Log("Deploying to Kubernetes...")
	cmd = exec.Command("kubectl", "apply", "-f", "test/k8s/deployment.yaml")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to deploy: %v\nOutput: %s", err, output)
	}

	// Wait for pod to be ready
	t.Log("Waiting for pod to be ready...")
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

PodReady:
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for pod to be ready")
		case <-ticker.C:
			cmd := exec.Command("kubectl", "get", "pods", "-l", "app=finfocus-aws-plugin", "-o", "jsonpath={.items[0].status.phase}")
			output, err := cmd.Output()
			if err != nil {
				continue
			}

			phase := strings.TrimSpace(string(output))
			t.Logf("Pod phase: %s", phase)

			if phase == "Running" {
				// Check readiness
				cmd := exec.Command("kubectl", "get", "pods", "-l", "app=finfocus-aws-plugin", "-o", "jsonpath={.items[0].status.conditions[?(@.type=='Ready')].status}")
				output, err := cmd.Output()
				if err == nil && strings.TrimSpace(string(output)) == "True" {
					t.Log("Pod is ready!")
					break PodReady
				}
			}
		}
	}

	// Port forward to check endpoints
	t.Log("Setting up port forwarding...")
	cmd = exec.Command("kubectl", "port-forward", "deployment/finfocus-aws-plugin", "8001:8001", "9090:9090")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start port forward: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()
	go func() {
		_ = cmd.Wait()
	}()

	// Wait a bit for port forwarding
	time.Sleep(5 * time.Second)

	// Check health endpoint via kubectl port-forward
	t.Log("Checking health via port forward...")
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get("http://localhost:8001/healthz")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read health response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Errorf("Health check returned status %d: %s", resp.StatusCode, body)
	}

	t.Log("Kubernetes integration test completed successfully")
}
