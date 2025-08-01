# Microservices Example

This example demonstrates logging in a microservices architecture using zlog's signal-based approach for distributed tracing and service coordination.

## Service Architecture

```go
package main

import (
    "context"
    "fmt"
    "math/rand"
    "time"

    "github.com/zoobzio/zlog"
)

// Distributed tracing signals
const (
    // Service lifecycle
    SERVICE_STARTING = "SERVICE_STARTING"
    SERVICE_READY    = "SERVICE_READY"
    SERVICE_STOPPING = "SERVICE_STOPPING"
    
    // Inter-service communication
    SERVICE_CALL_STARTED   = "SERVICE_CALL_STARTED"
    SERVICE_CALL_COMPLETED = "SERVICE_CALL_COMPLETED"
    SERVICE_CALL_FAILED    = "SERVICE_CALL_FAILED"
    
    // Circuit breaker events
    CIRCUIT_BREAKER_OPENED  = "CIRCUIT_BREAKER_OPENED"
    CIRCUIT_BREAKER_CLOSED  = "CIRCUIT_BREAKER_CLOSED"
    CIRCUIT_BREAKER_HALF_OPEN = "CIRCUIT_BREAKER_HALF_OPEN"
    
    // Business events
    ORDER_CREATED      = "ORDER_CREATED"
    ORDER_VALIDATED    = "ORDER_VALIDATED"
    ORDER_PROCESSED    = "ORDER_PROCESSED"
    ORDER_SHIPPED      = "ORDER_SHIPPED"
    ORDER_CANCELLED    = "ORDER_CANCELLED"
    
    PAYMENT_AUTHORIZED = "PAYMENT_AUTHORIZED"
    PAYMENT_CAPTURED   = "PAYMENT_CAPTURED"
    PAYMENT_FAILED     = "PAYMENT_FAILED"
    
    INVENTORY_RESERVED = "INVENTORY_RESERVED"
    INVENTORY_RELEASED = "INVENTORY_RELEASED"
    INVENTORY_DEPLETED = "INVENTORY_DEPLETED"
)

// Trace context for distributed tracing
type TraceContext struct {
    TraceID  string `json:"trace_id"`
    SpanID   string `json:"span_id"`
    ParentID string `json:"parent_id,omitempty"`
}

// Service represents a microservice
type Service struct {
    Name    string
    Version string
    Port    int
}

// Order Service
type OrderService struct {
    *Service
    paymentClient   *PaymentClient
    inventoryClient *InventoryClient
}

func main() {
    setupLogging()
    
    // Start services
    orderSvc := NewOrderService()
    
    // Simulate distributed order processing
    go simulateOrders(orderSvc)
    
    // Keep services running
    time.Sleep(30 * time.Second)
}

func setupLogging() {
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Setup distributed tracing
    setupDistributedTracing()
    
    // Setup service mesh monitoring
    setupServiceMeshMonitoring()
    
    // Setup business event aggregation
    setupBusinessEventAggregation()
}

func NewOrderService() *OrderService {
    svc := &Service{
        Name:    "order-service",
        Version: "1.0.0",
        Port:    8001,
    }
    
    zlog.Emit(SERVICE_STARTING, "Order service starting",
        zlog.String("service", svc.Name),
        zlog.String("version", svc.Version),
        zlog.Int("port", svc.Port))
    
    orderSvc := &OrderService{
        Service:         svc,
        paymentClient:   NewPaymentClient("http://localhost:8002"),
        inventoryClient: NewInventoryClient("http://localhost:8003"),
    }
    
    zlog.Emit(SERVICE_READY, "Order service ready",
        zlog.String("service", svc.Name))
    
    return orderSvc
}

// Order processing workflow
func (os *OrderService) ProcessOrder(ctx context.Context, orderID string, items []OrderItem) error {
    trace := extractTraceContext(ctx)
    
    // Create order
    zlog.Emit(ORDER_CREATED, "Order created",
        zlog.String("service", os.Name),
        zlog.String("order_id", orderID),
        zlog.String("trace_id", trace.TraceID),
        zlog.String("span_id", trace.SpanID),
        zlog.Int("item_count", len(items)))
    
    // Step 1: Reserve inventory
    for _, item := range items {
        err := os.inventoryClient.ReserveItem(ctx, item.ProductID, item.Quantity)
        if err != nil {
            zlog.Emit(ORDER_CANCELLED, "Order cancelled - inventory unavailable",
                zlog.String("service", os.Name),
                zlog.String("order_id", orderID),
                zlog.String("trace_id", trace.TraceID),
                zlog.String("product_id", item.ProductID),
                zlog.Err(err))
            return err
        }
    }
    
    // Step 2: Process payment
    total := calculateTotal(items)
    err := os.paymentClient.ProcessPayment(ctx, orderID, total)
    if err != nil {
        // Release inventory on payment failure
        for _, item := range items {
            os.inventoryClient.ReleaseItem(ctx, item.ProductID, item.Quantity)
        }
        
        zlog.Emit(ORDER_CANCELLED, "Order cancelled - payment failed",
            zlog.String("service", os.Name),
            zlog.String("order_id", orderID),
            zlog.String("trace_id", trace.TraceID),
            zlog.Float64("amount", total),
            zlog.Err(err))
        return err
    }
    
    // Step 3: Validate order
    zlog.Emit(ORDER_VALIDATED, "Order validated successfully",
        zlog.String("service", os.Name),
        zlog.String("order_id", orderID),
        zlog.String("trace_id", trace.TraceID),
        zlog.Float64("total_amount", total))
    
    zlog.Emit(ORDER_PROCESSED, "Order processed successfully",
        zlog.String("service", os.Name),
        zlog.String("order_id", orderID),
        zlog.String("trace_id", trace.TraceID),
        zlog.Float64("total_amount", total))
    
    return nil
}

// Payment Client
type PaymentClient struct {
    baseURL string
}

func NewPaymentClient(baseURL string) *PaymentClient {
    return &PaymentClient{baseURL: baseURL}
}

func (pc *PaymentClient) ProcessPayment(ctx context.Context, orderID string, amount float64) error {
    trace := extractTraceContext(ctx)
    childTrace := createChildSpan(trace, "payment-call")
    
    zlog.Emit(SERVICE_CALL_STARTED, "Payment service call started",
        zlog.String("service", "order-service"),
        zlog.String("target_service", "payment-service"),
        zlog.String("operation", "process_payment"),
        zlog.String("order_id", orderID),
        zlog.String("trace_id", trace.TraceID),
        zlog.String("span_id", childTrace.SpanID),
        zlog.String("parent_span_id", trace.SpanID),
        zlog.Float64("amount", amount))
    
    start := time.Now()
    
    // Simulate payment processing
    time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
    
    // Simulate occasional failures
    if rand.Float32() < 0.1 {
        duration := time.Since(start)
        
        zlog.Emit(SERVICE_CALL_FAILED, "Payment service call failed",
            zlog.String("service", "order-service"),
            zlog.String("target_service", "payment-service"),
            zlog.String("operation", "process_payment"),
            zlog.String("order_id", orderID),
            zlog.String("trace_id", trace.TraceID),
            zlog.String("span_id", childTrace.SpanID),
            zlog.Duration("duration", duration))
        
        zlog.Emit(PAYMENT_FAILED, "Payment processing failed",
            zlog.String("order_id", orderID),
            zlog.String("trace_id", trace.TraceID),
            zlog.Float64("amount", amount))
        
        return fmt.Errorf("payment gateway timeout")
    }
    
    duration := time.Since(start)
    
    zlog.Emit(SERVICE_CALL_COMPLETED, "Payment service call completed",
        zlog.String("service", "order-service"),
        zlog.String("target_service", "payment-service"),
        zlog.String("operation", "process_payment"),
        zlog.String("order_id", orderID),
        zlog.String("trace_id", trace.TraceID),
        zlog.String("span_id", childTrace.SpanID),
        zlog.Duration("duration", duration))
    
    zlog.Emit(PAYMENT_CAPTURED, "Payment captured successfully",
        zlog.String("order_id", orderID),
        zlog.String("trace_id", trace.TraceID),
        zlog.Float64("amount", amount))
    
    return nil
}

// Inventory Client
type InventoryClient struct {
    baseURL string
}

func NewInventoryClient(baseURL string) *InventoryClient {
    return &InventoryClient{baseURL: baseURL}
}

func (ic *InventoryClient) ReserveItem(ctx context.Context, productID string, quantity int) error {
    trace := extractTraceContext(ctx)
    childTrace := createChildSpan(trace, "inventory-reserve")
    
    zlog.Emit(SERVICE_CALL_STARTED, "Inventory service call started",
        zlog.String("service", "order-service"),
        zlog.String("target_service", "inventory-service"),
        zlog.String("operation", "reserve_item"),
        zlog.String("product_id", productID),
        zlog.String("trace_id", trace.TraceID),
        zlog.String("span_id", childTrace.SpanID),
        zlog.Int("quantity", quantity))
    
    start := time.Now()
    
    // Simulate inventory check
    time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
    
    // Simulate stock depletion
    if rand.Float32() < 0.05 {
        duration := time.Since(start)
        
        zlog.Emit(SERVICE_CALL_FAILED, "Inventory service call failed",
            zlog.String("service", "order-service"),
            zlog.String("target_service", "inventory-service"),
            zlog.String("operation", "reserve_item"),
            zlog.String("product_id", productID),
            zlog.String("trace_id", trace.TraceID),
            zlog.String("span_id", childTrace.SpanID),
            zlog.Duration("duration", duration))
        
        return fmt.Errorf("insufficient stock")
    }
    
    duration := time.Since(start)
    
    zlog.Emit(SERVICE_CALL_COMPLETED, "Inventory service call completed",
        zlog.String("service", "order-service"),
        zlog.String("target_service", "inventory-service"),
        zlog.String("operation", "reserve_item"),
        zlog.String("product_id", productID),
        zlog.String("trace_id", trace.TraceID),
        zlog.String("span_id", childTrace.SpanID),
        zlog.Duration("duration", duration))
    
    zlog.Emit(INVENTORY_RESERVED, "Inventory reserved successfully",
        zlog.String("product_id", productID),
        zlog.String("trace_id", trace.TraceID),
        zlog.Int("quantity", quantity))
    
    return nil
}

func (ic *InventoryClient) ReleaseItem(ctx context.Context, productID string, quantity int) error {
    trace := extractTraceContext(ctx)
    
    zlog.Emit(INVENTORY_RELEASED, "Inventory released",
        zlog.String("product_id", productID),
        zlog.String("trace_id", trace.TraceID),
        zlog.Int("quantity", quantity))
    
    return nil
}

// Types
type OrderItem struct {
    ProductID string  `json:"product_id"`
    Quantity  int     `json:"quantity"`
    Price     float64 `json:"price"`
}

// Utility functions
func extractTraceContext(ctx context.Context) *TraceContext {
    if trace, ok := ctx.Value("trace").(*TraceContext); ok {
        return trace
    }
    
    // Create new trace if none exists
    return &TraceContext{
        TraceID: generateTraceID(),
        SpanID:  generateSpanID(),
    }
}

func createChildSpan(parent *TraceContext, operation string) *TraceContext {
    return &TraceContext{
        TraceID:  parent.TraceID,
        SpanID:   generateSpanID(),
        ParentID: parent.SpanID,
    }
}

func generateTraceID() string {
    return fmt.Sprintf("trace_%d", time.Now().UnixNano())
}

func generateSpanID() string {
    return fmt.Sprintf("span_%d", time.Now().UnixNano())
}

func calculateTotal(items []OrderItem) float64 {
    total := 0.0
    for _, item := range items {
        total += item.Price * float64(item.Quantity)
    }
    return total
}

// Simulation
func simulateOrders(orderSvc *OrderService) {
    for i := 0; i < 10; i++ {
        orderID := fmt.Sprintf("order_%d", time.Now().UnixNano())
        
        // Create trace context
        trace := &TraceContext{
            TraceID: generateTraceID(),
            SpanID:  generateSpanID(),
        }
        
        ctx := context.WithValue(context.Background(), "trace", trace)
        
        items := []OrderItem{
            {ProductID: "widget-1", Quantity: 2, Price: 19.99},
            {ProductID: "widget-2", Quantity: 1, Price: 39.99},
        }
        
        err := orderSvc.ProcessOrder(ctx, orderID, items)
        if err != nil {
            zlog.Error("Order processing failed",
                zlog.String("order_id", orderID),
                zlog.String("trace_id", trace.TraceID),
                zlog.Err(err))
        }
        
        time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
    }
}

// Logging Setup
func setupDistributedTracing() {
    tracingSink := zlog.NewSink("distributed-tracing", func(ctx context.Context, event zlog.Event) error {
        // Send trace data to distributed tracing system (Jaeger, Zipkin, etc.)
        return nil
    })
    
    zlog.RouteSignal(SERVICE_CALL_STARTED, tracingSink)
    zlog.RouteSignal(SERVICE_CALL_COMPLETED, tracingSink)
    zlog.RouteSignal(SERVICE_CALL_FAILED, tracingSink)
}

func setupServiceMeshMonitoring() {
    meshSink := zlog.NewSink("service-mesh", func(ctx context.Context, event zlog.Event) error {
        // Send service mesh metrics to Istio, Linkerd, etc.
        return nil
    })
    
    zlog.RouteSignal(CIRCUIT_BREAKER_OPENED, meshSink)
    zlog.RouteSignal(CIRCUIT_BREAKER_CLOSED, meshSink)
    zlog.RouteSignal(SERVICE_CALL_FAILED, meshSink)
}

func setupBusinessEventAggregation() {
    businessSink := zlog.NewSink("business-events", func(ctx context.Context, event zlog.Event) error {
        // Aggregate business events for analytics
        return nil
    })
    
    zlog.RouteSignal(ORDER_CREATED, businessSink)
    zlog.RouteSignal(ORDER_PROCESSED, businessSink)
    zlog.RouteSignal(ORDER_CANCELLED, businessSink)
    zlog.RouteSignal(PAYMENT_CAPTURED, businessSink)
    zlog.RouteSignal(PAYMENT_FAILED, businessSink)
}
```

## Example Output

Distributed trace for a successful order:

```json
{"time":"2023-10-20T14:30:00Z","signal":"ORDER_CREATED","message":"Order created","service":"order-service","order_id":"order_1640123420000000001","trace_id":"trace_1640123420000000001","span_id":"span_1640123420000000001","item_count":2}

{"time":"2023-10-20T14:30:00Z","signal":"SERVICE_CALL_STARTED","message":"Inventory service call started","service":"order-service","target_service":"inventory-service","operation":"reserve_item","product_id":"widget-1","trace_id":"trace_1640123420000000001","span_id":"span_1640123420000000002","parent_span_id":"span_1640123420000000001","quantity":2}

{"time":"2023-10-20T14:30:00Z","signal":"SERVICE_CALL_COMPLETED","message":"Inventory service call completed","service":"order-service","target_service":"inventory-service","operation":"reserve_item","product_id":"widget-1","trace_id":"trace_1640123420000000001","span_id":"span_1640123420000000002","duration":"245ms"}

{"time":"2023-10-20T14:30:00Z","signal":"INVENTORY_RESERVED","message":"Inventory reserved successfully","product_id":"widget-1","trace_id":"trace_1640123420000000001","quantity":2}

{"time":"2023-10-20T14:30:01Z","signal":"SERVICE_CALL_STARTED","message":"Payment service call started","service":"order-service","target_service":"payment-service","operation":"process_payment","order_id":"order_1640123420000000001","trace_id":"trace_1640123420000000001","span_id":"span_1640123420000000003","parent_span_id":"span_1640123420000000001","amount":79.97}

{"time":"2023-10-20T14:30:01Z","signal":"PAYMENT_CAPTURED","message":"Payment captured successfully","order_id":"order_1640123420000000001","trace_id":"trace_1640123420000000001","amount":79.97}

{"time":"2023-10-20T14:30:02Z","signal":"ORDER_PROCESSED","message":"Order processed successfully","service":"order-service","order_id":"order_1640123420000000001","trace_id":"trace_1640123420000000001","total_amount":79.97}
```

This example demonstrates:

- **Distributed tracing**: Trace IDs and span IDs flowing through service calls
- **Service-to-service communication**: Detailed logging of inter-service calls
- **Business workflow tracking**: End-to-end visibility into order processing
- **Error correlation**: Failures traced across service boundaries
- **Performance monitoring**: Service call durations and bottleneck identification

The signal-based approach makes it easy to correlate events across services and build comprehensive distributed tracing and monitoring systems.