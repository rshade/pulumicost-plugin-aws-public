package main

import (
	"flag"
	"time"
)

// Config holds settings for the metrics aggregator server.
// StartPort and EndPort define the scraping port range, ListenAddr specifies
// the listen address for the HTTP server, and Timeout sets the per-scrape limit.
type Config struct {
	StartPort  int
	EndPort    int
	ListenAddr string
	Timeout    time.Duration
}

func parseConfig() *Config {
	config := &Config{}

	flag.IntVar(&config.StartPort, "start-port", 8001, "Starting port number to scrape")
	flag.IntVar(&config.EndPort, "end-port", 8012, "Ending port number to scrape")
	flag.StringVar(&config.ListenAddr, "listen", ":9090", "Address to listen on for metrics endpoint")
	flag.DurationVar(&config.Timeout, "timeout", 5*time.Second, "Timeout for scraping individual endpoints")

	flag.Parse()

	return config
}
