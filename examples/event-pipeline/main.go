// Package main demonstrates using zlog as a complete event processing pipeline.
//
// This example shows how zlog can serve as the central event bus for an application,
// handling everything from debug logs to business events to metrics and monitoring.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zoobzio/zlog"
)

// Business event signals for our e-commerce platform
const (
	// User events
	USER_SIGNUP       = zlog.Signal("USER_SIGNUP")
	USER_LOGIN        = zlog.Signal("USER_LOGIN") 
	USER_PROFILE_UPDATE = zlog.Signal("USER_PROFILE_UPDATE")

	// Product events
	PRODUCT_VIEWED    = zlog.Signal("PRODUCT_VIEWED")
	PRODUCT_SEARCHED  = zlog.Signal("PRODUCT_SEARCHED")
	REVIEW_POSTED     = zlog.Signal("REVIEW_POSTED")

	// Commerce events
	CART_UPDATED      = zlog.Signal("CART_UPDATED")
	CHECKOUT_STARTED  = zlog.Signal("CHECKOUT_STARTED")
	ORDER_PLACED      = zlog.Signal("ORDER_PLACED")
	PAYMENT_PROCESSED = zlog.Signal("PAYMENT_PROCESSED")
	ORDER_SHIPPED     = zlog.Signal("ORDER_SHIPPED")

	// System events
	CACHE_HIT         = zlog.Signal("CACHE_HIT")
	CACHE_MISS        = zlog.Signal("CACHE_MISS")
	API_CALLED        = zlog.Signal("API_CALLED")
	RATE_LIMITED      = zlog.Signal("RATE_LIMITED")
	SERVICE_HEALTH    = zlog.Signal("SERVICE_HEALTH")
)

// EventCorrelator tracks related events across the system.
type EventCorrelator struct {
	mu       sync.RWMutex
	sessions map[string][]zlog.Event
}

func NewEventCorrelator() *EventCorrelator {
	return &EventCorrelator{
		sessions: make(map[string][]zlog.Event),
	}
}

func (ec *EventCorrelator) AddEvent(sessionID string, event zlog.Event) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.sessions[sessionID] = append(ec.sessions[sessionID], event)
}

func (ec *EventCorrelator) GetSession(sessionID string) []zlog.Event {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.sessions[sessionID]
}

// MetricsAggregator collects and aggregates metrics from events.
type MetricsAggregator struct {
	counters sync.Map
	gauges   sync.Map
}

func (ma *MetricsAggregator) IncrementCounter(name string) {
	val, _ := ma.counters.LoadOrStore(name, new(int64))
	atomic.AddInt64(val.(*int64), 1)
}

func (ma *MetricsAggregator) SetGauge(name string, value float64) {
	ma.gauges.Store(name, value)
}

func (ma *MetricsAggregator) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	ma.counters.Range(func(key, value interface{}) bool {
		stats[key.(string)] = atomic.LoadInt64(value.(*int64))
		return true
	})
	
	ma.gauges.Range(func(key, value interface{}) bool {
		stats[key.(string)] = value.(float64)
		return true
	})
	
	return stats
}

// Global components
var (
	correlator = NewEventCorrelator()
	metrics    = &MetricsAggregator{}
)

// correlationSink adds events to the correlation engine.
var correlationSink = zlog.NewSink("correlation", func(ctx context.Context, event zlog.Event) error {
	// Extract session ID from fields
	for _, field := range event.Fields {
		if field.Key == "session_id" {
			if sessionID, ok := field.Value.(string); ok {
				correlator.AddEvent(sessionID, event)
			}
			break
		}
	}
	return nil
})

// metricsSink extracts metrics from events.
var metricsSink = zlog.NewSink("metrics", func(ctx context.Context, event zlog.Event) error {
	// Count all events by signal
	metrics.IncrementCounter(string(event.Signal))
	
	// Extract specific metrics
	for _, field := range event.Fields {
		switch field.Key {
		case "response_time":
			if d, ok := field.Value.(time.Duration); ok {
				metrics.SetGauge(string(event.Signal)+".response_time_ms", float64(d.Milliseconds()))
			}
		case "cart_value":
			if v, ok := field.Value.(float64); ok {
				metrics.SetGauge("cart.value", v)
			}
		case "order_total":
			if v, ok := field.Value.(float64); ok {
				metrics.SetGauge("order.total", v)
			}
		}
	}
	
	return nil
})

// businessAnalyticsSink processes business events for analytics.
var businessAnalyticsSink = zlog.NewSink("analytics", func(ctx context.Context, event zlog.Event) error {
	// In a real system, this would send to a data warehouse
	switch event.Signal {
	case PRODUCT_VIEWED, PRODUCT_SEARCHED:
		fmt.Printf("📊 [Analytics] User engagement: %s\n", event.Message)
	case ORDER_PLACED, PAYMENT_PROCESSED:
		fmt.Printf("💰 [Analytics] Revenue event: %s\n", event.Message)
	}
	return nil
})

// alertingSink handles critical events that need immediate attention.
var alertingSink = zlog.NewSink("alerting", func(ctx context.Context, event zlog.Event) error {
	fmt.Printf("🚨 [ALERT] %s: %s\n", event.Signal, event.Message)
	return nil
}).WithFilter(func(_ context.Context, event zlog.Event) bool {
	// Only alert on critical signals
	return event.Signal == RATE_LIMITED || event.Signal == zlog.ERROR || event.Signal == zlog.FATAL
})

// auditSink provides compliance logging.
var auditSink = zlog.NewSink("audit", func(ctx context.Context, event zlog.Event) error {
	// In production, write to immutable audit log
	if event.Signal == USER_LOGIN || event.Signal == ORDER_PLACED || event.Signal == PAYMENT_PROCESSED {
		fmt.Printf("📝 [Audit] %s at %s\n", event.Signal, event.Time.Format(time.RFC3339))
	}
	return nil
})

// setupPipeline configures the complete event routing.
func setupPipeline() {
	// All events go to correlation and metrics
	signals := []zlog.Signal{
		USER_SIGNUP, USER_LOGIN, USER_PROFILE_UPDATE,
		PRODUCT_VIEWED, PRODUCT_SEARCHED, REVIEW_POSTED,
		CART_UPDATED, CHECKOUT_STARTED, ORDER_PLACED,
		PAYMENT_PROCESSED, ORDER_SHIPPED,
		CACHE_HIT, CACHE_MISS, API_CALLED, RATE_LIMITED,
	}
	
	for _, signal := range signals {
		zlog.RouteSignal(signal, correlationSink)
		zlog.RouteSignal(signal, metricsSink)
	}
	
	// Business events to analytics
	businessSignals := []zlog.Signal{
		PRODUCT_VIEWED, PRODUCT_SEARCHED, CART_UPDATED,
		ORDER_PLACED, PAYMENT_PROCESSED,
	}
	for _, signal := range businessSignals {
		zlog.RouteSignal(signal, businessAnalyticsSink)
	}
	
	// Critical events to alerting
	zlog.RouteSignal(RATE_LIMITED, alertingSink)
	
	// Compliance events to audit
	zlog.RouteSignal(USER_LOGIN, auditSink)
	zlog.RouteSignal(ORDER_PLACED, auditSink)
	zlog.RouteSignal(PAYMENT_PROCESSED, auditSink)
	
	// Standard error logging
	zlog.EnableStandardLogging(zlog.INFO)
	zlog.RouteSignal(zlog.ERROR, alertingSink)
	
	// File sink for everything (with rotation)
	fileSink := zlog.NewRotatingFileSink("events.log", 10*1024*1024, 5).
		WithAsync() // Don't block on file I/O
		
	for _, signal := range signals {
		zlog.RouteSignal(signal, fileSink)
	}
}

// simulateUserSession simulates a complete user journey.
func simulateUserSession(userID, sessionID string) {
	// User logs in
	zlog.Emit(USER_LOGIN, "User logged in",
		zlog.String("user_id", userID),
		zlog.String("session_id", sessionID),
		zlog.Time("login_time", time.Now()),
	)
	
	// Search for products
	searches := []string{"laptop", "gaming laptop", "laptop accessories"}
	for _, query := range searches {
		zlog.Emit(PRODUCT_SEARCHED, "Product search",
			zlog.String("session_id", sessionID),
			zlog.String("query", query),
			zlog.Int("results_count", rand.Intn(50)+10),
		)
		time.Sleep(100 * time.Millisecond)
		
		// Simulate cache behavior
		if rand.Float32() > 0.3 {
			zlog.Emit(CACHE_HIT, "Search results from cache",
				zlog.String("session_id", sessionID),
				zlog.String("cache_key", "search:"+query),
			)
		} else {
			zlog.Emit(CACHE_MISS, "Search results from database",
				zlog.String("session_id", sessionID),
				zlog.String("cache_key", "search:"+query),
			)
		}
	}
	
	// View products
	products := []struct{ id, name string; price float64 }{
		{"prod-1", "Gaming Laptop Pro", 1299.99},
		{"prod-2", "Wireless Mouse", 59.99},
		{"prod-3", "Mechanical Keyboard", 149.99},
	}
	
	for _, product := range products {
		zlog.Emit(PRODUCT_VIEWED, "Product viewed",
			zlog.String("session_id", sessionID),
			zlog.String("product_id", product.id),
			zlog.String("product_name", product.name),
			zlog.Float64("price", product.price),
			zlog.Duration("view_duration", time.Duration(rand.Intn(30)+10)*time.Second),
		)
		time.Sleep(200 * time.Millisecond)
	}
	
	// Update cart
	cartValue := 0.0
	for i, product := range products[:rand.Intn(len(products))+1] {
		cartValue += product.price
		zlog.Emit(CART_UPDATED, "Item added to cart",
			zlog.String("session_id", sessionID),
			zlog.String("product_id", product.id),
			zlog.Float64("cart_value", cartValue),
			zlog.Int("cart_items", i+1),
		)
		time.Sleep(150 * time.Millisecond)
	}
	
	// Checkout process
	if cartValue > 0 {
		zlog.Emit(CHECKOUT_STARTED, "Checkout initiated",
			zlog.String("session_id", sessionID),
			zlog.Float64("cart_value", cartValue),
		)
		
		// Place order
		orderID := fmt.Sprintf("ORD-%06d", rand.Intn(1000000))
		zlog.Emit(ORDER_PLACED, "Order placed",
			zlog.String("session_id", sessionID),
			zlog.String("order_id", orderID),
			zlog.Float64("order_total", cartValue),
			zlog.String("user_id", userID),
		)
		
		// Process payment
		zlog.Emit(PAYMENT_PROCESSED, "Payment successful",
			zlog.String("session_id", sessionID),
			zlog.String("order_id", orderID),
			zlog.Float64("amount", cartValue),
			zlog.String("payment_method", "credit_card"),
		)
	}
}

// simulateAPITraffic generates API events.
func simulateAPITraffic() {
	endpoints := []string{"/api/products", "/api/users", "/api/orders", "/api/search"}
	
	for i := 0; i < 20; i++ {
		endpoint := endpoints[rand.Intn(len(endpoints))]
		responseTime := time.Duration(rand.Intn(200)+50) * time.Millisecond
		
		zlog.Emit(API_CALLED, "API request",
			zlog.String("endpoint", endpoint),
			zlog.String("method", "GET"),
			zlog.Duration("response_time", responseTime),
			zlog.Int("status", 200),
		)
		
		// Simulate rate limiting
		if i > 15 {
			zlog.Emit(RATE_LIMITED, "Rate limit exceeded",
				zlog.String("endpoint", endpoint),
				zlog.String("client_ip", "192.168.1.100"),
				zlog.Int("requests_count", i),
			)
		}
		
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	fmt.Println("=== Event Pipeline Example ===")
	fmt.Println("Demonstrating zlog as application event bus")
	fmt.Println()
	
	// Set up the complete pipeline
	setupPipeline()
	
	// Simulate multiple user sessions
	fmt.Println("--- Simulating User Sessions ---")
	var wg sync.WaitGroup
	
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			userID := fmt.Sprintf("user-%d", 1000+i)
			sessionID := fmt.Sprintf("session-%d", rand.Intn(10000))
			simulateUserSession(userID, sessionID)
		}(i)
		time.Sleep(500 * time.Millisecond)
	}
	
	// Simulate API traffic
	fmt.Println("\n--- Simulating API Traffic ---")
	simulateAPITraffic()
	
	wg.Wait()
	time.Sleep(500 * time.Millisecond) // Let async sinks finish
	
	// Show session correlation
	fmt.Println("\n--- Session Correlation Example ---")
	// Pick a session and show all events
	for sessionID, events := range correlator.sessions {
		if len(events) > 5 {
			fmt.Printf("\nSession %s journey (%d events):\n", sessionID, len(events))
			for i, event := range events {
				if i < 5 || i >= len(events)-2 {
					fmt.Printf("  %d. [%s] %s\n", i+1, event.Signal, event.Message)
				} else if i == 5 {
					fmt.Printf("  ... %d more events ...\n", len(events)-7)
				}
			}
			break
		}
	}
	
	// Show metrics summary
	fmt.Println("\n--- Metrics Summary ---")
	stats := metrics.GetStats()
	fmt.Println("Event counts:")
	for signal, count := range stats {
		if c, ok := count.(int64); ok && c > 0 {
			fmt.Printf("  %s: %d\n", signal, c)
		}
	}
	
	fmt.Println("\nGauges:")
	for name, value := range stats {
		if v, ok := value.(float64); ok {
			fmt.Printf("  %s: %.2f\n", name, v)
		}
	}
	
	// Clean up
	_ = os.Remove("events.log")
	
	fmt.Println("\n=== Example Complete ===")
	fmt.Println("Demonstrated:")
	fmt.Println("- Event correlation across sessions")
	fmt.Println("- Metrics aggregation")
	fmt.Println("- Business analytics")
	fmt.Println("- Alerting on critical events")
	fmt.Println("- Audit trail for compliance")
	fmt.Println("- Complete event pipeline with multiple sinks")
}