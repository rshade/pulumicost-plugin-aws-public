package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	config := parseConfig()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/metrics/aggregated", func(w http.ResponseWriter, r *http.Request) {
		aggregatedMetricsHandler(w, r, config)
	})

	log.Printf("Starting metrics aggregator on %s", config.ListenPort)
	log.Fatal(http.ListenAndServe(config.ListenPort, nil))
}

func aggregatedMetricsHandler(w http.ResponseWriter, r *http.Request, config *Config) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	var allMetrics strings.Builder

	for port := config.StartPort; port <= config.EndPort; port++ {
		metrics, err := fetchMetrics(ctx, port)
		if err != nil {
			log.Printf("Failed to fetch metrics from port %d: %v", port, err)
			continue
		}
		allMetrics.WriteString(metrics)
		allMetrics.WriteString("\n")
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(allMetrics.String()))
}

func fetchMetrics(ctx context.Context, port int) (string, error) {
	url := fmt.Sprintf("http://localhost:%d/metrics", port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
