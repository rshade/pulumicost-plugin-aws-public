//go:build integration

package integration

import (
	"bytes"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	imageName   = "finfocus-aws:test"
	testVersion = "v0.0.3"
)

func TestDockerImageBuildAndVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the Docker image

	t.Log("Building Docker image...")

	cmd := exec.Command("docker", "build", "--build-arg", "VERSION="+testVersion, "-t", imageName, "-f", "docker/Dockerfile", ".")

	cmd.Dir = ".."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Docker image: %v\nOutput: %s", err, output)
	}

	// Check image size
	t.Log("Checking image size...")
	cmd = exec.Command("docker", "image", "inspect", imageName, "--format", "{{.Size}}")
	sizeOutput, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get image size: %v", err)
	}
	sizeStr := strings.TrimSpace(string(sizeOutput))
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse image size: %v", err)
	}
	t.Logf("Image size: %d bytes", size)

	// Size should be reasonable (at least 1GB, less than 3GB)
	if size < 1*1024*1024*1024 || size > 3*1024*1024*1024 {
		t.Fatalf("Image size out of expected range: %d bytes", size)
	}

	// Run the container
	t.Log("Running container...")
	cmd = exec.Command("docker", "run", "-d", "--name", "test-aws", "-p", "8001-8012:8001-8012", "-p", "9090:9090", imageName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, output)
	}

	containerID := strings.TrimSpace(string(output))
	defer func() {
		exec.Command("docker", "rm", "-f", containerID).Run()
	}()

	// Wait for container to be healthy
	t.Log("Waiting for container health...")
	healthTimeout := time.After(2 * time.Minute)
	healthTicker := time.NewTicker(2 * time.Second)
	defer healthTicker.Stop()

	for {
		select {
		case <-healthTimeout:
			t.Fatal("Timeout waiting for container health")
		case <-healthTicker.C:
			inspectCmd := exec.Command("docker", "inspect", "--format", "{{.State.Health.Status}}", containerID)
			inspectOutput, err := inspectCmd.Output()
			if err != nil {
				continue
			}
			status := strings.TrimSpace(string(inspectOutput))
			if status == "healthy" || status == "" {
				if status == "" {
					inspectCmd = exec.Command("docker", "inspect", "--format", "{{.State.Status}}", containerID)
					inspectOutput, err = inspectCmd.Output()
					if err == nil && strings.TrimSpace(string(inspectOutput)) == "running" {
						break
					}
					continue
				}
				break
			}
		}
	}

	// Check health endpoint
	t.Log("Checking health endpoint...")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://localhost:8001/healthz")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Health check returned status %d", resp.StatusCode)
	}

	// Check metrics endpoint
	t.Log("Checking metrics endpoint...")
	resp, err = client.Get("http://localhost:9090/metrics")
	if err != nil {
		t.Fatalf("Metrics check failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Metrics check returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read metrics response: %v", err)
	}
	if !bytes.Contains(body, []byte("# HELP")) {
		t.Error("Metrics response does not contain Prometheus format")
	}

	// Check logs for region prefixes
	t.Log("Checking container logs...")
	cmd = exec.Command("docker", "logs", containerID)
	logOutput, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	logs := string(logOutput)
	if !strings.Contains(logs, `"region":"us-east-1"`) {
		t.Error("Logs do not contain injected region field")
	}

	t.Log("Docker integration test completed successfully")
}
