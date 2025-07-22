package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/zoobzio/zlog"
)

// TraceContext represents distributed tracing context.
type TraceContext struct {
	Baggage      map[string]string
	TraceID      string
	SpanID       string
	ParentSpanID string
}

// Service represents a microservice.
type Service struct {
	Name     string
	Version  string
	Instance string
}

// ServiceCall represents a call between services.
type ServiceCall struct {
	Error    error
	From     string
	To       string
	Method   string
	Duration time.Duration
}

func main() {
	// Enable logging with service identification
	zlog.EnableStandardLogging(os.Stderr)

	// Start distributed system simulation
	zlog.Info("Distributed system starting")

	system := &DistributedSystem{
		services: map[string]*Service{
			"api-gateway": {
				Name:     "api-gateway",
				Version:  "1.0.0",
				Instance: "gateway-1",
			},
			"user-service": {
				Name:     "user-service",
				Version:  "2.1.0",
				Instance: "user-3",
			},
			"order-service": {
				Name:     "order-service",
				Version:  "1.5.0",
				Instance: "order-2",
			},
			"notification-service": {
				Name:     "notification-service",
				Version:  "1.2.0",
				Instance: "notif-1",
			},
			"inventory-service": {
				Name:     "inventory-service",
				Version:  "3.0.0",
				Instance: "inv-4",
			},
		},
	}

	// Simulate distributed requests
	system.SimulateRequests()

	zlog.Info("Distributed system simulation complete")
}

type DistributedSystem struct {
	services map[string]*Service
}

func (ds *DistributedSystem) SimulateRequests() {
	// Simulate different distributed scenarios

	// Scenario 1: Simple order flow
	ds.SimulateOrderFlow()

	// Scenario 2: Complex multi-service interaction
	ds.SimulateComplexFlow()

	// Scenario 3: Failed distributed transaction
	ds.SimulateFailedTransaction()

	// Scenario 4: Parallel service calls
	ds.SimulateParallelCalls()
}

func (ds *DistributedSystem) SimulateOrderFlow() {
	// Create root trace
	trace := TraceContext{
		TraceID: generateTraceID(),
		SpanID:  generateSpanID(),
		Baggage: map[string]string{
			"user_id":     "user_123",
			"session_id":  "sess_456",
			"request_id":  "req_789",
			"environment": "production",
		},
	}

	// API Gateway receives request
	gatewaySpan := ds.startSpan("api-gateway", trace, "POST /api/orders")

	// Call User Service
	userSpan := ds.callService("api-gateway", "user-service", gatewaySpan, "GetUser")
	ds.endSpan("user-service", userSpan, nil)

	// Call Order Service
	orderSpan := ds.callService("api-gateway", "order-service", gatewaySpan, "CreateOrder")

	// Order Service calls Inventory
	invSpan := ds.callService("order-service", "inventory-service", orderSpan, "CheckStock")
	ds.endSpan("inventory-service", invSpan, nil)

	// Order Service calls Notification
	notifSpan := ds.callService("order-service", "notification-service", orderSpan, "SendOrderConfirmation")
	ds.endSpan("notification-service", notifSpan, nil)

	ds.endSpan("order-service", orderSpan, nil)
	ds.endSpan("api-gateway", gatewaySpan, nil)
}

func (ds *DistributedSystem) SimulateComplexFlow() {
	trace := TraceContext{
		TraceID: generateTraceID(),
		SpanID:  generateSpanID(),
		Baggage: map[string]string{
			"user_id":  "user_456",
			"feature":  "bulk_order",
			"priority": "high",
		},
	}

	// Complex flow with multiple service interactions
	rootSpan := ds.startSpan("api-gateway", trace, "POST /api/bulk-orders")

	// Parallel calls to multiple services
	var wg sync.WaitGroup
	wg.Add(3)

	// Check user permissions
	go func() {
		defer wg.Done()
		span := ds.callService("api-gateway", "user-service", rootSpan, "CheckPermissions")
		time.Sleep(time.Duration(rand.Intn(50)+20) * time.Millisecond)
		ds.endSpan("user-service", span, nil)
	}()

	// Validate inventory for multiple items
	go func() {
		defer wg.Done()
		span := ds.callService("api-gateway", "inventory-service", rootSpan, "BulkCheckStock")

		// Inventory makes multiple internal calls
		for i := 0; i < 3; i++ {
			itemSpan := ds.callService("inventory-service", "inventory-service", span,
				fmt.Sprintf("CheckItem_%d", i))
			time.Sleep(time.Duration(rand.Intn(30)+10) * time.Millisecond)
			ds.endSpan("inventory-service", itemSpan, nil)
		}

		ds.endSpan("inventory-service", span, nil)
	}()

	// Pre-calculate pricing
	go func() {
		defer wg.Done()
		span := ds.callService("api-gateway", "order-service", rootSpan, "CalculateBulkPricing")
		time.Sleep(time.Duration(rand.Intn(60)+40) * time.Millisecond)
		ds.endSpan("order-service", span, nil)
	}()

	wg.Wait()
	ds.endSpan("api-gateway", rootSpan, nil)
}

func (ds *DistributedSystem) SimulateFailedTransaction() {
	trace := TraceContext{
		TraceID: generateTraceID(),
		SpanID:  generateSpanID(),
		Baggage: map[string]string{
			"user_id":      "user_789",
			"payment_type": "credit_card",
		},
	}

	rootSpan := ds.startSpan("api-gateway", trace, "POST /api/checkout")

	// Successful user check
	userSpan := ds.callService("api-gateway", "user-service", rootSpan, "ValidateUser")
	ds.endSpan("user-service", userSpan, nil)

	// Order creation starts
	orderSpan := ds.callService("api-gateway", "order-service", rootSpan, "ProcessCheckout")

	// Inventory check fails
	invSpan := ds.callService("order-service", "inventory-service", orderSpan, "ReserveStock")
	ds.endSpan("inventory-service", invSpan, fmt.Errorf("insufficient stock for item SKU_12345"))

	// Rollback notification
	notifSpan := ds.callService("order-service", "notification-service", orderSpan, "SendFailureNotification")
	ds.endSpan("notification-service", notifSpan, nil)

	ds.endSpan("order-service", orderSpan, fmt.Errorf("checkout failed: insufficient stock"))
	ds.endSpan("api-gateway", rootSpan, fmt.Errorf("checkout failed"))
}

func (ds *DistributedSystem) SimulateParallelCalls() {
	trace := TraceContext{
		TraceID: generateTraceID(),
		SpanID:  generateSpanID(),
		Baggage: map[string]string{
			"user_id": "user_999",
			"action":  "dashboard_load",
		},
	}

	rootSpan := ds.startSpan("api-gateway", trace, "GET /api/dashboard")

	// Fan-out to multiple services in parallel
	services := []string{"user-service", "order-service", "notification-service", "inventory-service"}
	var wg sync.WaitGroup

	for _, service := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()

			span := ds.callService("api-gateway", svc, rootSpan, "GetDashboardData")

			// Simulate varying response times
			duration := time.Duration(rand.Intn(100)+50) * time.Millisecond
			time.Sleep(duration)

			// 10% chance of error
			var err error
			if rand.Float32() < 0.1 {
				err = fmt.Errorf("timeout calling %s", svc)
			}

			ds.endSpan(svc, span, err)
		}(service)
	}

	wg.Wait()
	ds.endSpan("api-gateway", rootSpan, nil)
}

// Helper methods for distributed tracing

func (ds *DistributedSystem) startSpan(serviceName string, trace TraceContext, operation string) TraceContext {
	service := ds.services[serviceName]

	fields := []zlog.Field{
		zlog.String("service", service.Name),
		zlog.String("version", service.Version),
		zlog.String("instance", service.Instance),
		zlog.String("trace_id", trace.TraceID),
		zlog.String("span_id", trace.SpanID),
		zlog.String("operation", operation),
	}

	// Add parent span if exists
	if trace.ParentSpanID != "" {
		fields = append(fields, zlog.String("parent_span_id", trace.ParentSpanID))
	}

	// Add baggage items
	for k, v := range trace.Baggage {
		fields = append(fields, zlog.String("baggage."+k, v))
	}

	zlog.Info("Span started", fields...)

	return trace
}

func (ds *DistributedSystem) callService(from, to string, parentTrace TraceContext, method string) TraceContext {
	// Create child span for the service call
	childTrace := TraceContext{
		TraceID:      parentTrace.TraceID,
		SpanID:       generateSpanID(),
		ParentSpanID: parentTrace.SpanID,
		Baggage:      parentTrace.Baggage,
	}

	// Log the service call
	zlog.Info("Service call initiated",
		zlog.String("from_service", from),
		zlog.String("to_service", to),
		zlog.String("method", method),
		zlog.String("trace_id", childTrace.TraceID),
		zlog.String("span_id", childTrace.SpanID),
		zlog.String("parent_span_id", childTrace.ParentSpanID),
	)

	// Start span in the called service
	ds.startSpan(to, childTrace, method)

	return childTrace
}

func (ds *DistributedSystem) endSpan(serviceName string, trace TraceContext, err error) {
	service := ds.services[serviceName]

	fields := []zlog.Field{
		zlog.String("service", service.Name),
		zlog.String("trace_id", trace.TraceID),
		zlog.String("span_id", trace.SpanID),
		zlog.Duration("duration", time.Duration(rand.Intn(100)+10)*time.Millisecond),
	}

	if err != nil {
		fields = append(fields,
			zlog.Err(err),
			zlog.String("status", "error"),
		)
		zlog.Error("Span completed with error", fields...)
	} else {
		fields = append(fields, zlog.String("status", "success"))
		zlog.Info("Span completed", fields...)
	}
}

// Correlation helpers

func generateTraceID() string {
	return fmt.Sprintf("trace_%d_%d", time.Now().UnixNano(), rand.Intn(1000000))
}

func generateSpanID() string {
	return fmt.Sprintf("span_%d_%d", time.Now().UnixNano()/1000, rand.Intn(10000))
}

// Context propagation helpers

func InjectTraceContext(ctx context.Context, trace TraceContext) context.Context {
	return context.WithValue(ctx, "trace", trace)
}

func ExtractTraceContext(ctx context.Context) (TraceContext, bool) {
	trace, ok := ctx.Value("trace").(TraceContext)
	return trace, ok
}

// TracingSink could be used to send traces to a tracing backend.
type TracingSink struct {
	serviceName string
}

func NewTracingSink(serviceName string) *TracingSink {
	return &TracingSink{serviceName: serviceName}
}

func (s *TracingSink) Write(event zlog.Event) error {
	// In a real implementation, this would:
	// 1. Extract trace context from the event
	// 2. Convert to OpenTelemetry span
	// 3. Send to tracing backend (Jaeger, Zipkin, etc.)

	// Check if trace_id is present
	hasTraceID := false
	var traceID string
	for _, field := range event.Fields {
		if field.Key == "trace_id" {
			hasTraceID = true
			if val, ok := field.Value.(string); ok {
				traceID = val
			}
			break
		}
	}

	if !hasTraceID {
		// In production, this would create a new trace
		// For now, just log that trace is missing
		fmt.Fprintf(os.Stderr, "[TracingSink] Event missing trace_id: %s\n", event.Message)
	} else {
		// In production, send to tracing backend
		// For now, just validate it's a valid trace
		if traceID == "" {
			fmt.Fprintf(os.Stderr, "[TracingSink] Empty trace_id in event: %s\n", event.Message)
		}
	}

	return nil
}

func (s *TracingSink) Name() string {
	return fmt.Sprintf("tracing:%s", s.serviceName)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
