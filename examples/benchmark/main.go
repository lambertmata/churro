package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type BenchmarkConfig struct {
	URL         string
	Requests    int
	Concurrency int
	Method      string
	Body        string
	Headers     map[string]string
}

type BenchmarkResult struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	TotalTime       time.Duration
	AverageTime     time.Duration
	RequestsPerSec  float64
	MinTime         time.Duration
	MaxTime         time.Duration
	Percentiles     map[int]time.Duration
}

func main() {
	var (
		url         = flag.String("url", "http://localhost:8888/", "URL to benchmark")
		requests    = flag.Int("n", 1000, "Number of requests to make")
		concurrency = flag.Int("c", 10, "Number of concurrent requests")
		method      = flag.String("m", "GET", "HTTP method")
		body        = flag.String("d", "", "Request body")
	)
	flag.Parse()

	config := BenchmarkConfig{
		URL:         *url,
		Requests:    *requests,
		Concurrency: *concurrency,
		Method:      *method,
		Body:        *body,
	}

	fmt.Printf("Benchmarking %s\n", config.URL)
	fmt.Printf("Requests: %d, Concurrency: %d\n\n", config.Requests, config.Concurrency)

	result := runBenchmark(config)
	printResults(result)
}

func runBenchmark(config BenchmarkConfig) BenchmarkResult {
	var wg sync.WaitGroup
	var mu sync.Mutex

	results := make([]time.Duration, 0, config.Requests)
	successCount := 0
	failedCount := 0

	// Channel to control concurrency
	semaphore := make(chan struct{}, config.Concurrency)

	startTime := time.Now()

	for i := 0; i < config.Requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			requestStart := time.Now()
			success := makeRequest(config)
			requestDuration := time.Since(requestStart)

			mu.Lock()
			results = append(results, requestDuration)
			if success {
				successCount++
			} else {
				failedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	// Calculate statistics
	return calculateStats(results, successCount, failedCount, totalTime)
}

func makeRequest(config BenchmarkConfig) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var bodyReader io.Reader
	if config.Body != "" {
		bodyReader = strings.NewReader(config.Body)
	}

	req, err := http.NewRequest(config.Method, config.URL, bodyReader)
	if err != nil {
		return false
	}

	if config.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Read and discard response body
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func calculateStats(durations []time.Duration, success, failed int, totalTime time.Duration) BenchmarkResult {
	if len(durations) == 0 {
		return BenchmarkResult{}
	}

	// Sort durations for percentile calculation
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	var total time.Duration
	min := durations[0]
	max := durations[0]

	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := total / time.Duration(len(durations))
	rps := float64(success) / totalTime.Seconds()

	percentiles := map[int]time.Duration{
		50: durations[len(durations)*50/100],
		90: durations[len(durations)*90/100],
		95: durations[len(durations)*95/100],
		99: durations[len(durations)*99/100],
	}

	return BenchmarkResult{
		TotalRequests:   len(durations),
		SuccessRequests: success,
		FailedRequests:  failed,
		TotalTime:       totalTime,
		AverageTime:     avg,
		RequestsPerSec:  rps,
		MinTime:         min,
		MaxTime:         max,
		Percentiles:     percentiles,
	}
}

func printResults(result BenchmarkResult) {
	fmt.Printf("Results:\n")
	fmt.Printf("========\n")
	fmt.Printf("Total requests:      %d\n", result.TotalRequests)
	fmt.Printf("Successful requests: %d\n", result.SuccessRequests)
	fmt.Printf("Failed requests:     %d\n", result.FailedRequests)
	fmt.Printf("Total time:          %v\n", result.TotalTime)
	fmt.Printf("\nTiming:\n")
	fmt.Printf("Requests per second: %.2f\n", result.RequestsPerSec)
	fmt.Printf("Average time:        %v\n", result.AverageTime)
	fmt.Printf("Min time:            %v\n", result.MinTime)
	fmt.Printf("Max time:            %v\n", result.MaxTime)
	fmt.Printf("\nPercentiles:\n")
	fmt.Printf("50%% (median):        %v\n", result.Percentiles[50])
	fmt.Printf("90%%:                 %v\n", result.Percentiles[90])
	fmt.Printf("95%%:                 %v\n", result.Percentiles[95])
	fmt.Printf("99%%:                 %v\n", result.Percentiles[99])
}
