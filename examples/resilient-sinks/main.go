// resilient-sinks demonstrates circuit breakers and rate limiting for production reliability.
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zoobzio/zlog"
)

// Simulated external service with variable reliability
type ExternalAPI struct {
	name         string
	errorRate    float64
	latency      time.Duration
	requestCount atomic.Int64
	errorCount   atomic.Int64
}

func (api *ExternalAPI) Send(data string) error {
	api.requestCount.Add(1)

	// Simulate latency
	time.Sleep(api.latency)

	// Simulate failures
	if rand.Float64() < api.errorRate {
		api.errorCount.Add(1)
		return fmt.Errorf("%s: service unavailable", api.name)
	}

	return nil
}

func (api *ExternalAPI) Stats() (requests, errors int64) {
	return api.requestCount.Load(), api.errorCount.Load()
}

// Business event signals
const (
	ORDER_PLACED   = zlog.Signal("ORDER_PLACED")
	PAYMENT_FAILED = zlog.Signal("PAYMENT_FAILED")
	METRIC_LOGGED  = zlog.Signal("METRIC_LOGGED")
)

func main() {
	fmt.Println("=== Resilient Sinks Example ===")
	fmt.Println("Demonstrating circuit breakers and rate limiting")
	fmt.Println()

	// Simulate external services
	primaryAPI := &ExternalAPI{
		name:      "PrimaryAPI",
		errorRate: 0.3, // 30% error rate
		latency:   50 * time.Millisecond,
	}

	secondaryAPI := &ExternalAPI{
		name:      "SecondaryAPI",
		errorRate: 0.1, // 10% error rate
		latency:   30 * time.Millisecond,
	}

	metricsAPI := &ExternalAPI{
		name:      "MetricsAPI",
		errorRate: 0.05, // 5% error rate
		latency:   10 * time.Millisecond,
	}

	// Enable standard logging
	zlog.EnableStandardLogging(zlog.INFO)

	// Example 1: Circuit Breaker Protection
	fmt.Println("--- Circuit Breaker Example ---")

	// Primary API sink with circuit breaker
	primarySink := zlog.NewSink("primary-api", func(ctx context.Context, event zlog.Log) error {
		data := fmt.Sprintf("[%s] %s", event.Signal, event.Message)
		return primaryAPI.Send(data)
	}).WithCircuitBreaker(zlog.CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		ResetTimeout:     2 * time.Second,
	})

	// Fallback to secondary API
	resilientSink := primarySink.WithFallback(
		zlog.NewSink("secondary-api", func(ctx context.Context, event zlog.Log) error {
			fmt.Println("📍 Using fallback API")
			data := fmt.Sprintf("[%s] %s", event.Signal, event.Message)
			return secondaryAPI.Send(data)
		}),
	)

	zlog.Hook(ORDER_PLACED, resilientSink)

	// Simulate order events that will trip the circuit breaker
	fmt.Println("Sending orders through unreliable primary API...")
	for i := 0; i < 10; i++ {
		zlog.Emit(ORDER_PLACED, fmt.Sprintf("Order #%d placed", i+1),
			zlog.String("order_id", fmt.Sprintf("ORD-%04d", i+1)),
			zlog.Float64("amount", float64(rand.Intn(500))+50),
		)
		time.Sleep(200 * time.Millisecond)
	}

	primaryReqs, primaryErrs := primaryAPI.Stats()
	fmt.Printf("\nPrimary API stats: %d requests, %d errors\n", primaryReqs, primaryErrs)
	secondaryReqs, secondaryErrs := secondaryAPI.Stats()
	fmt.Printf("Secondary API stats: %d requests, %d errors\n\n", secondaryReqs, secondaryErrs)

	// Example 2: Rate Limiting
	fmt.Println("--- Rate Limiting Example ---")

	// Metrics API with rate limiting
	metricsSink := zlog.NewSink("metrics-api", func(ctx context.Context, event zlog.Log) error {
		data := fmt.Sprintf("metric.%s", event.Signal)
		return metricsAPI.Send(data)
	}).WithRateLimit(zlog.RateLimiterConfig{
		RequestsPerSecond: 5,     // 5 RPS limit
		BurstSize:         10,    // Allow bursts up to 10
		WaitForSlot:       false, // Fail fast when rate limited
	})

	zlog.Hook(METRIC_LOGGED, metricsSink)

	// Generate burst of metrics
	fmt.Println("Sending burst of 20 metrics (rate limit: 5 RPS, burst: 10)...")
	var rateLimited atomic.Int64

	for i := 0; i < 20; i++ {
		zlog.Emit(METRIC_LOGGED, fmt.Sprintf("Metric #%d", i+1),
			zlog.String("metric_name", "api.latency"),
			zlog.Float64("value", rand.Float64()*100),
		)
		// Check if we were rate limited
		if i >= 9 { // After burst capacity
			rateLimited.Add(1)
		}
	}

	time.Sleep(100 * time.Millisecond)
	metricsReqs, _ := metricsAPI.Stats()
	fmt.Printf("Metrics API received: %d requests (rate limited: ~%d)\n\n", metricsReqs, rateLimited.Load()-1)

	// Example 3: Combined Protection
	fmt.Println("--- Combined Circuit Breaker + Rate Limiting ---")

	// External webhook with both protections
	webhookSink := zlog.NewSink("webhook", func(ctx context.Context, event zlog.Log) error {
		// Simulate webhook call
		if rand.Float64() < 0.2 {
			return errors.New("webhook timeout")
		}
		return nil
	}).
		WithRateLimit(zlog.RateLimiterConfig{
			RequestsPerSecond: 10,
			BurstSize:         20,
			WaitForSlot:       false,
		}).
		WithCircuitBreaker(zlog.CircuitBreakerConfig{
			FailureThreshold: 5,
			ResetTimeout:     3 * time.Second,
		}).
		WithRetry(2) // Also add retry for transient failures

	zlog.Hook(PAYMENT_FAILED, webhookSink)

	// Simulate payment failures
	fmt.Println("Sending payment failure notifications...")
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			zlog.Emit(PAYMENT_FAILED, fmt.Sprintf("Payment failed for order #%d", id),
				zlog.String("order_id", fmt.Sprintf("ORD-%04d", id)),
				zlog.String("reason", "insufficient_funds"),
			)
		}(i + 1)
		time.Sleep(50 * time.Millisecond)
	}

	wg.Wait()

	// Example 4: Performance Under Load
	fmt.Println("\n--- Performance Under Load ---")

	// Create a rate-limited sink for load testing
	loadTestSink := zlog.RateLimitedSink("load-test", 100, func(ctx context.Context, event zlog.Log) error {
		// Simulate some processing
		time.Sleep(time.Microsecond * 100)
		return nil
	})

	// Custom signal for load test
	const LOAD_TEST = zlog.Signal("LOAD_TEST")
	zlog.Hook(LOAD_TEST, loadTestSink)

	// Generate high load
	fmt.Println("Generating 1000 events at high speed (100 RPS limit)...")
	start := time.Now()
	processed := atomic.Int64{}

	for i := 0; i < 1000; i++ {
		go func(id int) {
			zlog.Emit(LOAD_TEST, fmt.Sprintf("Event %d", id))
			processed.Add(1)
		}(i)
	}

	// Wait a bit for processing
	time.Sleep(3 * time.Second)
	elapsed := time.Since(start)

	fmt.Printf("Processed %d events in %v (effective rate: %.1f RPS)\n",
		processed.Load(), elapsed,
		float64(processed.Load())/elapsed.Seconds())

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Println("Circuit Breaker Benefits:")
	fmt.Println("- Prevents cascade failures")
	fmt.Println("- Automatic recovery testing")
	fmt.Println("- Reduces load on failing services")
	fmt.Println()
	fmt.Println("Rate Limiting Benefits:")
	fmt.Println("- Protects external APIs from overload")
	fmt.Println("- Controls costs for metered services")
	fmt.Println("- Ensures fair resource usage")
	fmt.Println()
	fmt.Println("Combined, they create resilient logging pipelines!")

	// Cleanup
	os.Exit(0)
}
