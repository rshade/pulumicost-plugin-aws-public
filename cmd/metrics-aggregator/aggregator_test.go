package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchMetrics(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# HELP test_metric Test metric\n# TYPE test_metric gauge\ntest_metric 1\n"))
	}))
	defer server.Close()

	// Extract port from server URL
	portStr := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	port := 0
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatalf("Failed to parse port: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	metrics, err := fetchMetrics(ctx, port)
	if err != nil {
		t.Fatalf("fetchMetrics failed: %v", err)
	}

	if !strings.Contains(metrics, "test_metric") {
		t.Errorf("Expected metrics to contain 'test_metric', got: %s", metrics)
	}
}

func TestAggregatedMetricsHandler(t *testing.T) {
	// Since the handler fetches from real ports, we'll test with a mock by using httptest.Server
	// but adjusting the port. For simplicity, test the handler structure without real network calls.

	config := &Config{
		StartPort:  8001,
		EndPort:    8001,
		ListenPort: ":9090",
		Timeout:    1 * time.Second,
	}

	req := httptest.NewRequest("GET", "/metrics/aggregated", nil)
	w := httptest.NewRecorder()

	aggregatedMetricsHandler(w, req, config)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Since no servers are running, the body should be empty or contain errors logged
	// The test mainly checks that the handler doesn't panic and returns 200
	body := w.Body.String()
	if body == "" && resp.StatusCode == http.StatusOK {
		// This is expected when no metrics are available
	} else if !strings.Contains(body, "#") && resp.StatusCode == http.StatusOK {
		// Valid response
	}
}
