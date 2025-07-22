package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/zoobzio/zlog"
)

// RequestContext holds request-specific information.
type RequestContext struct {
	RequestID string
	UserID    string
	Method    string
	Path      string
	TraceID   string // For distributed tracing
}

// contextKey is a type for context keys.
type contextKey string

const requestContextKey = contextKey("request")

func main() {
	// Enable standard logging
	zlog.EnableStandardLogging(os.Stderr)

	// Start the server
	zlog.Info("Server started", zlog.Int("port", 8080))

	// Simulate handling multiple requests
	server := &Server{
		db:    &Database{},
		cache: &Cache{},
	}

	// Process several requests to show correlation
	requests := []RequestContext{
		{
			RequestID: generateRequestID(),
			UserID:    "user123",
			Method:    "GET",
			Path:      "/api/users",
			TraceID:   generateTraceID(),
		},
		{
			RequestID: generateRequestID(),
			UserID:    "user456",
			Method:    "POST",
			Path:      "/api/orders",
			TraceID:   generateTraceID(),
		},
		{
			RequestID: generateRequestID(),
			UserID:    "user789",
			Method:    "GET",
			Path:      "/api/products/42",
			TraceID:   generateTraceID(),
		},
	}

	for _, req := range requests {
		ctx := context.WithValue(context.Background(), requestContextKey, req)
		server.HandleRequest(ctx)
		time.Sleep(100 * time.Millisecond) // Space out requests
	}

	zlog.Info("Server shutdown")
}

type Server struct {
	db    *Database
	cache *Cache
}

func (s *Server) HandleRequest(ctx context.Context) {
	start := time.Now()
	req := getRequestContext(ctx)

	// Log request start with all context
	logWithRequest(req, zlog.Info, "Request started",
		zlog.String("method", req.Method),
		zlog.String("path", req.Path),
		zlog.String("user_id", req.UserID),
		zlog.String("trace_id", req.TraceID),
	)

	// Simulate request processing
	switch req.Path {
	case "/api/users":
		s.handleUsers(ctx)
	case "/api/orders":
		s.handleOrders(ctx)
	default:
		s.handleProduct(ctx)
	}

	// Log request completion
	duration := time.Since(start)
	status := 200
	if rand.Float32() < 0.1 { // 10% error rate
		status = 500
	}

	logWithRequest(req, zlog.Info, "Request completed",
		zlog.Int("status", status),
		zlog.Duration("duration", duration),
		zlog.Int("duration_ms", int(duration.Milliseconds())),
	)
}

func (s *Server) handleUsers(ctx context.Context) {
	req := getRequestContext(ctx)

	// Check cache first
	cacheKey := "users:all"
	if data := s.cache.Get(ctx, cacheKey); data != nil {
		logWithRequest(req, zlog.Info, "Cache hit",
			zlog.String("key", cacheKey),
			zlog.Int("size_bytes", len(data)),
		)
		return
	}

	// Cache miss - query database
	logWithRequest(req, zlog.Info, "Cache miss",
		zlog.String("key", cacheKey),
	)

	users := s.db.QueryUsers(ctx)

	// Cache the result
	s.cache.Set(ctx, cacheKey, users)
}

func (s *Server) handleOrders(ctx context.Context) {
	req := getRequestContext(ctx)

	// Validate the order
	logWithRequest(req, zlog.Info, "Validating order",
		zlog.String("step", "validation"),
	)

	// Process payment (might fail)
	if err := processPayment(ctx); err != nil {
		logWithRequest(req, zlog.Error, "Payment failed",
			zlog.Err(err),
			zlog.String("step", "payment"),
		)
		return
	}

	// Create order in database
	orderID := s.db.CreateOrder(ctx)

	logWithRequest(req, zlog.Info, "Order created",
		zlog.String("order_id", orderID),
		zlog.String("step", "completion"),
	)
}

func (s *Server) handleProduct(ctx context.Context) {
	req := getRequestContext(ctx)

	// Extract product ID from path
	productID := "42" // Simplified

	logWithRequest(req, zlog.Info, "Fetching product",
		zlog.String("product_id", productID),
	)

	product := s.db.GetProduct(ctx, productID)
	if product == nil {
		logWithRequest(req, zlog.Warn, "Product not found",
			zlog.String("product_id", productID),
		)
	}
}

// Database simulates database operations.
type Database struct{}

func (db *Database) QueryUsers(ctx context.Context) []byte {
	req := getRequestContext(ctx)

	query := "SELECT id, name, email FROM users WHERE active = true"
	logWithRequest(req, zlog.Info, "Database query",
		zlog.String("query", query),
		zlog.String("table", "users"),
		zlog.Int("rows_returned", 42),
	)

	// Simulate query time
	time.Sleep(20 * time.Millisecond)

	return []byte(`[{"id":"user123","name":"Alice"},{"id":"user456","name":"Bob"}]`)
}

func (db *Database) CreateOrder(ctx context.Context) string {
	req := getRequestContext(ctx)

	orderID := fmt.Sprintf("order_%d", rand.Intn(10000))

	logWithRequest(req, zlog.Info, "Database insert",
		zlog.String("table", "orders"),
		zlog.String("order_id", orderID),
	)

	return orderID
}

func (db *Database) GetProduct(ctx context.Context, productID string) []byte {
	req := getRequestContext(ctx)

	query := fmt.Sprintf("SELECT * FROM products WHERE id = '%s'", productID)
	logWithRequest(req, zlog.Info, "Database query",
		zlog.String("query", query),
		zlog.String("table", "products"),
	)

	if productID == "42" {
		return []byte(`{"id":"42","name":"Ultimate Widget","price":99.99}`)
	}
	return nil
}

// Cache simulates a cache service.
type Cache struct {
	data map[string][]byte
}

func (c *Cache) Get(ctx context.Context, key string) []byte {
	if c.data == nil {
		c.data = make(map[string][]byte)
	}
	return c.data[key]
}

func (c *Cache) Set(ctx context.Context, key string, value []byte) {
	req := getRequestContext(ctx)

	if c.data == nil {
		c.data = make(map[string][]byte)
	}
	c.data[key] = value

	logWithRequest(req, zlog.Info, "Cache set",
		zlog.String("key", key),
		zlog.Int("size_bytes", len(value)),
		zlog.Int("ttl_seconds", 300),
	)
}

// Helper functions

func processPayment(ctx context.Context) error {
	req := getRequestContext(ctx)

	// Simulate payment processing
	logWithRequest(req, zlog.Info, "Processing payment",
		zlog.String("gateway", "stripe"),
		zlog.String("currency", "USD"),
	)

	if rand.Float32() < 0.2 { // 20% failure rate
		return fmt.Errorf("payment gateway timeout")
	}

	return nil
}

// logWithRequest adds request context to all log entries.
func logWithRequest(req RequestContext, logFunc func(string, ...zlog.Field), msg string, fields ...zlog.Field) {
	// Always include request_id as the first field for easy filtering
	contextFields := []zlog.Field{
		zlog.String("request_id", req.RequestID),
	}

	// Append any additional fields
	contextFields = append(contextFields, fields...)

	// Call the log function
	logFunc(msg, contextFields...)
}

// getRequestContext retrieves request context from context.Context.
func getRequestContext(ctx context.Context) RequestContext {
	if req, ok := ctx.Value(requestContextKey).(RequestContext); ok {
		return req
	}
	// Return empty context if not found
	return RequestContext{RequestID: "unknown"}
}

// generateRequestID creates a unique request ID.
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), rand.Intn(100000))
}

// generateTraceID creates a trace ID for distributed tracing.
func generateTraceID() string {
	return fmt.Sprintf("trace_%d_%d", time.Now().UnixNano(), rand.Intn(100000))
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
