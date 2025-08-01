// Package main demonstrates building custom sinks for zlog.
//
// This example shows how to create sinks that integrate with external
// systems like metrics collectors, message queues, and databases.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zoobzio/zlog"
)

// MetricsCollector simulates a metrics backend like Prometheus or StatsD.
type MetricsCollector struct {
	counters map[string]*int64
	gauges   map[string]float64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters: make(map[string]*int64),
		gauges:   make(map[string]float64),
	}
}

func (m *MetricsCollector) IncrementCounter(name string, value int64) {
	if _, exists := m.counters[name]; !exists {
		var v int64
		m.counters[name] = &v
	}
	atomic.AddInt64(m.counters[name], value)
}

func (m *MetricsCollector) SetGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *MetricsCollector) Print() {
	fmt.Println("\n📊 Metrics Summary:")
	for name, value := range m.counters {
		fmt.Printf("  Counter %s: %d\n", name, atomic.LoadInt64(value))
	}
	for name, value := range m.gauges {
		fmt.Printf("  Gauge %s: %.2f\n", name, value)
	}
}

// MessageQueue simulates a message queue like Kafka or RabbitMQ.
type MessageQueue struct {
	topic    string
	messages []string
}

func NewMessageQueue(topic string) *MessageQueue {
	return &MessageQueue{
		topic:    topic,
		messages: make([]string, 0),
	}
}

func (mq *MessageQueue) Publish(message string) error {
	mq.messages = append(mq.messages, message)
	fmt.Printf("📤 [MQ:%s] Published: %s\n", mq.topic, message)
	return nil
}

// NewMetricsSink creates a sink that extracts metrics from events.
func NewMetricsSink(collector *MetricsCollector) *zlog.Sink {
	return zlog.NewSink("metrics", func(ctx context.Context, event zlog.Event) error {
		// Count events by signal
		metricName := strings.ToLower(string(event.Signal))
		collector.IncrementCounter(metricName+".count", 1)

		// Extract numeric fields as metrics
		for _, field := range event.Fields {
			switch field.Key {
			case "duration":
				if d, ok := field.Value.(time.Duration); ok {
					collector.SetGauge(metricName+".duration_ms", float64(d.Milliseconds()))
				}
			case "amount":
				if v, ok := field.Value.(float64); ok {
					collector.SetGauge(metricName+".amount", v)
				}
			case "count", "items":
				if v, ok := field.Value.(int); ok {
					collector.IncrementCounter(metricName+"."+field.Key, int64(v))
				}
			}
		}

		return nil
	})
}

// NewMessageQueueSink creates a sink that publishes events to a message queue.
func NewMessageQueueSink(queue *MessageQueue, signals ...zlog.Signal) *zlog.Sink {
	// Convert signals to a map for fast lookup
	signalSet := make(map[zlog.Signal]bool)
	for _, s := range signals {
		signalSet[s] = true
	}

	return zlog.NewSink("message-queue", func(ctx context.Context, event zlog.Event) error {
		// Only publish specific signals
		if len(signalSet) > 0 && !signalSet[event.Signal] {
			return nil
		}

		// Create a simplified message format
		message := map[string]interface{}{
			"timestamp": event.Time.Unix(),
			"signal":    string(event.Signal),
			"message":   event.Message,
		}

		// Add selected fields
		for _, field := range event.Fields {
			// Only include serializable fields
			switch field.Key {
			case "user_id", "order_id", "amount", "status":
				message[field.Key] = field.Value
			}
		}

		// Serialize and publish
		data, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		return queue.Publish(string(data))
	})
}

// NewDatabaseAuditSink creates a sink that would write to a database.
func NewDatabaseAuditSink() *zlog.Sink {
	return zlog.NewSink("database-audit", func(ctx context.Context, event zlog.Event) error {
		// In a real implementation, this would use database/sql or an ORM
		
		// Simulate building an SQL insert
		var fields []string
		var values []interface{}
		
		fields = append(fields, "timestamp", "signal", "message")
		values = append(values, event.Time, string(event.Signal), event.Message)

		// Extract audit-relevant fields
		for _, field := range event.Fields {
			switch field.Key {
			case "user_id", "ip", "action", "resource", "result":
				fields = append(fields, field.Key)
				values = append(values, field.Value)
			}
		}

		// Simulate the insert
		fmt.Printf("🗄️  [DB] INSERT INTO audit_log (%s) VALUES (%v)\n", 
			strings.Join(fields, ", "), values)

		return nil
	})
}

// NewConditionalSink creates a sink that only processes events matching a condition.
func NewConditionalSink(name string, condition func(zlog.Event) bool, handler func(zlog.Event)) *zlog.Sink {
	return zlog.NewSink(name, func(ctx context.Context, event zlog.Event) error {
		if condition(event) {
			handler(event)
		}
		return nil
	})
}

// NewBatchingSink creates a sink that batches events before processing.
type BatchingSink struct {
	events   []zlog.Event
	maxBatch int
	handler  func([]zlog.Event) error
}

func NewBatchingSink(maxBatch int, handler func([]zlog.Event) error) *zlog.Sink {
	bs := &BatchingSink{
		events:   make([]zlog.Event, 0, maxBatch),
		maxBatch: maxBatch,
		handler:  handler,
	}

	return zlog.NewSink("batching", func(ctx context.Context, event zlog.Event) error {
		bs.events = append(bs.events, event)
		
		if len(bs.events) >= bs.maxBatch {
			// Process the batch
			err := bs.handler(bs.events)
			// Clear the batch
			bs.events = bs.events[:0]
			return err
		}
		
		return nil
	})
}

// Define some signals
const (
	API_REQUEST    = zlog.Signal("API_REQUEST")
	PAYMENT_EVENT  = zlog.Signal("PAYMENT_EVENT")
	USER_ACTION    = zlog.Signal("USER_ACTION")
	SYSTEM_METRIC  = zlog.Signal("SYSTEM_METRIC")
)

func main() {
	fmt.Println("=== Custom Sink Example ===")
	fmt.Println("Demonstrating various custom sink patterns")
	fmt.Println()

	// Create our external systems
	metrics := NewMetricsCollector()
	orderQueue := NewMessageQueue("orders")
	
	// Create custom sinks
	metricsSink := NewMetricsSink(metrics)
	queueSink := NewMessageQueueSink(orderQueue, PAYMENT_EVENT)
	auditSink := NewDatabaseAuditSink()
	
	// Create a conditional sink for high-value transactions
	highValueSink := NewConditionalSink("high-value", 
		func(e zlog.Event) bool {
			for _, field := range e.Fields {
				if field.Key == "amount" {
					if v, ok := field.Value.(float64); ok && v > 1000 {
						return true
					}
				}
			}
			return false
		},
		func(e zlog.Event) {
			fmt.Printf("💰 [HIGH VALUE] %s: %s\n", e.Signal, e.Message)
		},
	)

	// Create a batching sink
	batchSink := NewBatchingSink(3, func(events []zlog.Event) error {
		fmt.Printf("📦 [BATCH] Processing %d events\n", len(events))
		for _, e := range events {
			fmt.Printf("   - %s: %s\n", e.Signal, e.Message)
		}
		return nil
	})

	// Set up routing
	zlog.RouteSignal(API_REQUEST, metricsSink)
	zlog.RouteSignal(PAYMENT_EVENT, metricsSink)
	zlog.RouteSignal(PAYMENT_EVENT, queueSink)
	zlog.RouteSignal(PAYMENT_EVENT, highValueSink)
	zlog.RouteSignal(USER_ACTION, auditSink)
	zlog.RouteSignal(SYSTEM_METRIC, batchSink)

	// Also use standard logging
	zlog.EnableStandardLogging(zlog.INFO)

	// Generate some events
	fmt.Println("--- Simulating Events ---")
	
	// API requests
	for i := 0; i < 3; i++ {
		zlog.Emit(API_REQUEST, "API request handled",
			zlog.String("endpoint", "/api/users"),
			zlog.Int("status", 200),
			zlog.Duration("duration", time.Duration(50+i*10)*time.Millisecond),
		)
		time.Sleep(100 * time.Millisecond)
	}

	// Payment events
	amounts := []float64{99.99, 1500.00, 250.00, 5000.00}
	for i, amount := range amounts {
		zlog.Emit(PAYMENT_EVENT, "Payment processed",
			zlog.String("order_id", fmt.Sprintf("ORD-%d", 1000+i)),
			zlog.Float64("amount", amount),
			zlog.String("status", "success"),
		)
		time.Sleep(100 * time.Millisecond)
	}

	// User actions
	zlog.Emit(USER_ACTION, "User permission changed",
		zlog.String("user_id", "user123"),
		zlog.String("action", "grant_role"),
		zlog.String("resource", "admin_panel"),
		zlog.String("result", "success"),
	)

	// System metrics (batched)
	for i := 0; i < 5; i++ {
		zlog.Emit(SYSTEM_METRIC, fmt.Sprintf("Metric %d", i+1),
			zlog.Int("value", 100+i*10),
		)
	}

	// Show metrics summary
	time.Sleep(200 * time.Millisecond)
	metrics.Print()

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("Custom sinks demonstrated:")
	fmt.Println("- Metrics extraction")
	fmt.Println("- Message queue publishing")  
	fmt.Println("- Database audit logging")
	fmt.Println("- Conditional processing")
	fmt.Println("- Event batching")
}