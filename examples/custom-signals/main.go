// Package main demonstrates using custom signals for domain-specific events.
//
// Instead of forcing business events into severity levels, zlog lets you
// define signals that match your application's actual event types.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/zoobzio/zlog"
)

// Define domain-specific signals for an e-commerce platform.
// These represent actual business events, not severity levels.
const (
	// User lifecycle events
	USER_REGISTERED  = zlog.Signal("USER_REGISTERED")
	USER_LOGIN       = zlog.Signal("USER_LOGIN")
	USER_LOGOUT      = zlog.Signal("USER_LOGOUT")
	PASSWORD_CHANGED = zlog.Signal("PASSWORD_CHANGED")

	// Commerce events
	PRODUCT_VIEWED    = zlog.Signal("PRODUCT_VIEWED")
	CART_UPDATED      = zlog.Signal("CART_UPDATED")
	ORDER_PLACED      = zlog.Signal("ORDER_PLACED")
	PAYMENT_PROCESSED = zlog.Signal("PAYMENT_PROCESSED")
	PAYMENT_FAILED    = zlog.Signal("PAYMENT_FAILED")
	ORDER_SHIPPED     = zlog.Signal("ORDER_SHIPPED")

	// System events
	CACHE_HIT        = zlog.Signal("CACHE_HIT")
	CACHE_MISS       = zlog.Signal("CACHE_MISS")
	API_RATE_LIMITED = zlog.Signal("API_RATE_LIMITED")
	FRAUD_DETECTED   = zlog.Signal("FRAUD_DETECTED")
)

// auditSink handles events that need audit trail.
var auditSink = zlog.NewSink("audit", func(ctx context.Context, event zlog.Log) error {
	// In a real app, this would write to an audit log file or database
	// Notice how we can access context values for distributed tracing
	traceID := ctx.Value("trace_id")
	if traceID != nil {
		fmt.Printf("[AUDIT] [trace:%s] %s: %s\n", traceID, event.Signal, event.Message)
	} else {
		fmt.Printf("[AUDIT] %s: %s\n", event.Signal, event.Message)
	}
	return nil
})

// metricsSink extracts metrics from events.
var metricsSink = zlog.NewSink("metrics", func(ctx context.Context, event zlog.Log) error {
	// In a real app, this would send to Prometheus, StatsD, etc.
	switch event.Signal {
	case ORDER_PLACED, PAYMENT_PROCESSED:
		for _, field := range event.Data {
			if field.Key == "amount" {
				fmt.Printf("[METRICS] %s.amount: %.2f\n", event.Signal, field.Value)
			}
		}
	case CACHE_HIT, CACHE_MISS:
		fmt.Printf("[METRICS] cache.%s: 1\n", event.Signal)
	}
	return nil
})

// alertSink handles critical events that need immediate attention.
var alertSink = zlog.NewSink("alerts", func(ctx context.Context, event zlog.Log) error {
	// In a real app, this would send to PagerDuty, Slack, etc.
	fmt.Printf("[ALERT] ⚠️  %s: %s\n", event.Signal, event.Message)

	// Show all fields for context
	data := make(map[string]interface{})
	for _, field := range event.Data {
		data[field.Key] = field.Value
	}
	jsonData, _ := json.MarshalIndent(data, "         ", "  ")
	fmt.Printf("%s\n", jsonData)
	return nil
})

// analyticsFileSink writes events to a file for analytics processing.
var analyticsFileSink = zlog.NewRotatingFileSink(
	"analytics.log",
	10*1024*1024, // 10MB files
	3,            // Keep 3 files
).WithAsync() // Don't block on file I/O

// setupRouting configures which signals go to which sinks.
func setupRouting() {
	// User events go to audit trail (using variadic RouteSignal)
	zlog.RouteSignal(USER_REGISTERED, auditSink)
	zlog.RouteSignal(USER_LOGIN, auditSink)
	zlog.RouteSignal(USER_LOGOUT, auditSink)
	zlog.RouteSignal(PASSWORD_CHANGED, auditSink)

	// Commerce events go to multiple destinations (now more concise!)
	zlog.RouteSignal(ORDER_PLACED, auditSink, metricsSink, analyticsFileSink)
	zlog.RouteSignal(PAYMENT_PROCESSED, auditSink, metricsSink, analyticsFileSink)

	// Failed payments need alerts
	zlog.RouteSignal(PAYMENT_FAILED, auditSink, alertSink)

	// Cache events only go to metrics, but sample to reduce volume
	// Only process 10% of cache hits (they're high volume)
	sampledCacheHitSink := metricsSink.WithSampling(0.1)
	zlog.RouteSignal(CACHE_HIT, sampledCacheHitSink)
	zlog.RouteSignal(CACHE_MISS, metricsSink) // Process all misses

	// Security events need immediate attention
	zlog.RouteSignal(FRAUD_DETECTED, auditSink, alertSink)

	// Product analytics - sample product views to reduce load
	// Process 25% of product views (high volume event)
	sampledViewSink := analyticsFileSink.WithSampling(0.25)
	zlog.RouteSignal(PRODUCT_VIEWED, sampledViewSink)
	zlog.RouteSignal(CART_UPDATED, analyticsFileSink) // Process all cart updates

	// Also enable standard logging for errors
	zlog.EnableStandardLogging(zlog.ERROR)
}

// simulateUserJourney simulates a user shopping session with distributed tracing.
func simulateUserJourney(userID string) {
	// Create a trace context for this user session
	traceID := fmt.Sprintf("trace-%d", rand.Intn(10000))
	sessionCtx := context.WithValue(context.Background(), "trace_id", traceID)
	sessionCtx = context.WithValue(sessionCtx, "user_id", userID)

	// Set context for this goroutine - all subsequent Emit calls will use this context
	zlog.SetContext(sessionCtx)
	defer zlog.ClearContext()

	// User logs in
	zlog.Emit(USER_LOGIN, "User logged in",
		zlog.String("user_id", userID),
		zlog.String("ip", "192.168.1.100"),
		zlog.Time("login_time", time.Now()),
	)

	// Browse products
	products := []string{"laptop", "mouse", "keyboard", "monitor"}
	for i := 0; i < rand.Intn(5)+1; i++ {
		product := products[rand.Intn(len(products))]
		zlog.Emit(PRODUCT_VIEWED, "Product viewed",
			zlog.String("user_id", userID),
			zlog.String("product_id", product),
			zlog.Duration("view_duration", time.Duration(rand.Intn(30))*time.Second),
		)
		time.Sleep(100 * time.Millisecond)
	}

	// Update cart
	zlog.Emit(CART_UPDATED, "Items added to cart",
		zlog.String("user_id", userID),
		zlog.Int("item_count", rand.Intn(3)+1),
		zlog.Float64("cart_value", float64(rand.Intn(1000))+99.99),
	)

	// Place order
	orderID := fmt.Sprintf("ORD-%d", rand.Intn(10000))
	amount := float64(rand.Intn(500)) + 50.99

	zlog.Emit(ORDER_PLACED, "Order placed",
		zlog.String("user_id", userID),
		zlog.String("order_id", orderID),
		zlog.Float64("amount", amount),
		zlog.Int("item_count", rand.Intn(5)+1),
	)

	// Process payment
	if rand.Float32() > 0.1 { // 90% success rate
		zlog.Emit(PAYMENT_PROCESSED, "Payment successful",
			zlog.String("user_id", userID),
			zlog.String("order_id", orderID),
			zlog.Float64("amount", amount),
			zlog.String("payment_method", "credit_card"),
		)
	} else {
		zlog.Emit(PAYMENT_FAILED, "Payment failed",
			zlog.String("user_id", userID),
			zlog.String("order_id", orderID),
			zlog.Float64("amount", amount),
			zlog.String("reason", "insufficient_funds"),
		)
	}
}

// simulateCacheOperations shows system-level events.
func simulateCacheOperations() {
	cacheKeys := []string{"user:123", "product:laptop", "session:abc", "config:app"}

	// Generate more events to show sampling effect
	fmt.Printf("Generating 50 cache events (70%% hits, 30%% misses)...\n")
	fmt.Printf("Cache hits are sampled at 10%%, misses are not sampled\n")

	hitCount := 0
	missCount := 0

	for i := 0; i < 50; i++ {
		key := cacheKeys[rand.Intn(len(cacheKeys))]

		if rand.Float32() > 0.3 { // 70% hit rate
			hitCount++
			zlog.Emit(CACHE_HIT, "Cache hit",
				zlog.String("key", key),
				zlog.Duration("latency", time.Duration(rand.Intn(5))*time.Microsecond),
			)
		} else {
			missCount++
			zlog.Emit(CACHE_MISS, "Cache miss",
				zlog.String("key", key),
				zlog.Duration("fetch_time", time.Duration(rand.Intn(100))*time.Millisecond),
			)
		}

		if i < 10 {
			time.Sleep(50 * time.Millisecond) // Slow down first few for visibility
		}
	}

	fmt.Printf("Generated %d hits (expecting ~%d in metrics) and %d misses\n",
		hitCount, hitCount/10, missCount)
}

// simulateFraudDetection shows security events.
func simulateFraudDetection() {
	zlog.Emit(FRAUD_DETECTED, "Suspicious payment pattern detected",
		zlog.String("user_id", "user789"),
		zlog.String("reason", "multiple_failed_payments"),
		zlog.Int("failed_attempts", 5),
		zlog.Float64("total_amount", 5432.10),
		zlog.String("action_taken", "account_locked"),
	)
}

func main() {
	fmt.Println("=== Custom Signals Example ===")
	fmt.Println("Demonstrating domain-specific event routing with context propagation")
	fmt.Println()

	// Set up signal routing
	setupRouting()

	// Clean up analytics file
	defer os.Remove("analytics.log")

	// Simulate various business events
	fmt.Println("--- User Shopping Journey (with distributed tracing context) ---")
	simulateUserJourney("user123")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n--- Cache Operations ---")
	simulateCacheOperations()

	fmt.Println("\n--- Security Event ---")
	simulateFraudDetection()

	// Show traditional errors still work
	fmt.Println("\n--- Standard Error Logging ---")
	zlog.Error("Database connection timeout",
		zlog.String("host", "db.example.com"),
		zlog.Duration("timeout", 30*time.Second),
	)

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("Notice how different signals were routed to different sinks!")
	fmt.Println("The user journey events include trace IDs from context propagation.")
	fmt.Println("Check analytics.log for file output.")
}
