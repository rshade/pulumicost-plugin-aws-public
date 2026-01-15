package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

func main() {
	config := parseConfig()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/metrics/aggregated", func(w http.ResponseWriter, r *http.Request) {
		aggregatedMetricsHandler(w, r, config)
	})

	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}

	shutdownDone := make(chan struct{})
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		<-signalChan

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Shutdown failed")
		}
		close(shutdownDone)
	}()

	log.Info().Str("addr", config.ListenAddr).Msg("Starting metrics aggregator")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed")
	}
	<-shutdownDone
}

func aggregatedMetricsHandler(w http.ResponseWriter, r *http.Request, config *Config) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	var allMetrics strings.Builder

	for port := config.StartPort; port <= config.EndPort; port++ {
		metrics, err := fetchMetrics(ctx, port)
		if err != nil {
			log.Error().Err(err).Int("port", port).Msg("Failed to fetch metrics")
			continue
		}
		if _, err := allMetrics.WriteString(metrics); err != nil {
			log.Error().Err(err).Msg("Failed to append metrics")
			return
		}
		if _, err := allMetrics.WriteString("\n"); err != nil {
			log.Error().Err(err).Msg("Failed to append metrics newline")
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(allMetrics.String())); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
		return
	}
}

func fetchMetrics(ctx context.Context, port int) (string, error) {
	url := fmt.Sprintf("http://localhost:%d/metrics", port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
