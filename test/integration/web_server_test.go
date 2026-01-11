//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the plugin binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "plugin-server")

	// We build the main package. assuming we are in test/integration, so main is ../../cmd/pulumicost-plugin-aws-public
	rootDir, err := filepath.Abs("../..")
	require.NoError(t, err)

	t.Log("Building plugin binary...")
	cmdBuild := exec.Command("go", "build", "-tags", "region_use1", "-o", binaryPath, "./cmd/pulumicost-plugin-aws-public")
	cmdBuild.Dir = rootDir
	output, err := cmdBuild.CombinedOutput()
	require.NoError(t, err, "Build failed: %s", string(output))

	// Run the plugin
	t.Log("Starting plugin server...")
	cmd := exec.Command(binaryPath)
	// Enable web serving
	cmd.Env = append(os.Environ(), "PULUMICOST_PLUGIN_WEB_ENABLED=true")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		_ = cmd.Process.Kill()
	}()

	// Wait for port
	var port string
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.After(5 * time.Second)

		for {
			select {
			case <-done:
				return
			case <-timeout:
				return
			case <-ticker.C:
				out := stdout.String()
				if strings.Contains(out, "PORT=") {
					re := regexp.MustCompile(`PORT=(\d+)`)
					matches := re.FindStringSubmatch(out)
					if len(matches) > 1 {
						port = matches[1]
						close(done)
						return
					}
				}
			}
		}
	}()

	select {
	case <-done:
		t.Logf("Plugin listening on port %s", port)
	case <-time.After(6 * time.Second):
		t.Fatalf("Timed out waiting for port. Stdout: %s, Stderr: %s", stdout.String(), stderr.String())
	}

	// Make HTTP Request
	url := fmt.Sprintf("http://localhost:%s/pulumicost.v1.CostSourceService/GetProjectedCost", port)
	reqBody := []byte(`{
		"resource": {
			"provider": "aws",
			"resource_type": "ec2",
			"sku": "t3.micro",
			"region": "us-east-1"
		}
	}`)

	t.Logf("Sending POST request to %s", url)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Response Status: %s", resp.Status)
	t.Logf("Response Body: %s", string(body))

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Fallback pricing should return a cost for t3.micro
	assert.Contains(t, string(body), "costPerMonth")
	assert.Contains(t, string(body), "USD")

	// Test GetPluginInfo
	infoUrl := fmt.Sprintf("http://localhost:%s/pulumicost.v1.CostSourceService/GetPluginInfo", port)
	t.Logf("Sending POST request to %s", infoUrl)
	// Empty JSON body for GetPluginInfo
	respInfo, err := http.Post(infoUrl, "application/json", bytes.NewBuffer([]byte("{}")))
	require.NoError(t, err)
	defer respInfo.Body.Close()

	bodyInfo, err := io.ReadAll(respInfo.Body)
	require.NoError(t, err)

	t.Logf("Response Status: %s", respInfo.Status)
	t.Logf("Response Body: %s", string(bodyInfo))

	assert.Equal(t, http.StatusOK, respInfo.StatusCode)
	assert.Contains(t, string(bodyInfo), "pulumicost-plugin-aws-public")
}
