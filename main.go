package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Endpoint struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

type DomainStats struct {
	Success int
	Failure int
	Total   int
	Domain  string
}

var stats = make(map[string]*DomainStats)

// shared HTTP client
var httpClient = &http.Client{
	Timeout: 500 * time.Millisecond,
}

// refactored checkHealth to split up the function and improve readability (it was getting pretty long)

func prepareBody(raw string) (io.Reader, error) {

	byteSlice := []byte(raw)
	if json.Valid(byteSlice) {
		return bytes.NewReader(byteSlice), nil
	}
	marshalled, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}
	return bytes.NewReader(marshalled), nil
}

func buildRequest(ctx context.Context, ep Endpoint, body io.Reader) (*http.Request, error) {

	req, err := http.NewRequestWithContext(ctx, ep.Method, ep.URL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range ep.Headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func recordOutcome(ep Endpoint, resp *http.Response, err error) {

	domain := extractDomain(ep.URL)
	stats[domain].Total++

	switch {
	case err != nil && errors.Is(err, context.DeadlineExceeded):
		stats[domain].Failure++
		log.Printf("⏱️ timeout (500ms) for %s\n", ep.URL)

	case err != nil:
		stats[domain].Failure++
		log.Printf("❌ request error for %s: %v\n", ep.URL, err)

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		stats[domain].Success++
		log.Printf("✅ available: %s\n", ep.URL)

	default:
		stats[domain].Failure++
		log.Printf("❌ unavailable (%d): %s\n", resp.StatusCode, ep.URL)
	}
}

func checkHealth(parentCtx context.Context, ep Endpoint) {

	timeout := 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	bodyReader, err := prepareBody(ep.Body)
	if err != nil {
		log.Printf("🔴 [%s] bad body: %v\n", ep.Name, err)
		return
	}

	req, err := buildRequest(ctx, ep, bodyReader)
	if err != nil {
		log.Printf("🔴 [%s] build request: %v\n", ep.Name, err)
		return
	}

	resp, err := httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	recordOutcome(ep, resp, err)
}

func extractDomain(rawUrl string) string {

	// add a feature to ignore ports
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		// url is malformed (shouldn't happen tho)
		return err.Error()
	}
	// this determines hostname regardless if ports are specified
	domain := parsedUrl.Hostname()
	return domain
}

func monitorEndpoints(ctx context.Context, endpoints []Endpoint) {
	for _, endpoint := range endpoints {
		domain := extractDomain(endpoint.URL)
		if stats[domain] == nil {
			stats[domain] = &DomainStats{}
		}
	}

	// refactored this using a ticker instead of the explicit sleep
	// it's a little safer than using the explicit sleep

	sleepTime := time.NewTicker(15 * time.Second)
	defer sleepTime.Stop()

	// gotta make a fan-in channel...this was to solve the problem of it not running at start (was waiting 15 seconds and
	// then starting). It was better than the alternative of adding another loop on top of the outer for range.
	tickChan := make(chan time.Time, 1)
	tickChan <- time.Now()

	// forward all ticker events into the tick channel
	go func() {
		for t := range sleepTime.C {
			tickChan <- t
		}
	}()

	// run immediately and then every 15 seconds
	for range tickChan {
		for _, endpoint := range endpoints {
			log.Printf("Checking %v endpoint...\n", endpoint.URL)
			checkHealth(ctx, endpoint)
		}
		logResults()
	}

}

func logResults() {

	// use tablewriter for table output synopsis
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Domain", "Success", "Total", "Availability"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for domain, stat := range stats {
		pct := math.Round(100*float64(stat.Success)/float64(stat.Total)*100) / 100 // standard percentage formatting (two decimals)
		table.Append([]string{
			domain,
			fmt.Sprintf("%d", stat.Success),
			fmt.Sprintf("%d", stat.Total),
			fmt.Sprintf("%.2f%%", pct),
		})
	}

	table.Render()

}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("🔴 Usage: go run main.go <config_file>")
	}

	filePath := os.Args[1]
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("🔴 Error reading file:", err)
	}

	// add top level context
	ctx := context.Background()
	var endpoints []Endpoint
	if err := yaml.Unmarshal(data, &endpoints); err != nil {
		log.Fatal("🔴 Error parsing YAML:", err)
	}

	monitorEndpoints(ctx, endpoints)
}
