package integration

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

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
		exec.Command("kind", "delete", "cluster", "--name", clusterName).Run()
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
					goto podReady
				}
			}
		}
	}

podReady:
	// Port forward to check endpoints
	t.Log("Setting up port forwarding...")
	cmd = exec.Command("kubectl", "port-forward", "deployment/finfocus-aws-plugin", "8001:8001", "9090:9090")
	go func() {
		cmd.Run() // Run in background
	}()
	defer cmd.Process.Kill()

	// Wait a bit for port forwarding
	time.Sleep(5 * time.Second)

	// Check health endpoint via kubectl port-forward
	t.Log("Checking health via port forward...")
	cmd = exec.Command("curl", "-f", "http://localhost:8001/healthz")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Health check failed: %v\nOutput: %s", err, output)
	}

	t.Log("Kubernetes integration test completed successfully")
}
