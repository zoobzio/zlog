package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/zoobzio/zlog"
)

func main() {
	// Detect environment
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	// Configure logging based on environment
	switch env {
	case "production":
		configureProduction()
	default:
		configureDevelopment()
	}

	// Log startup
	zlog.Info("Application started",
		zlog.String("environment", env),
		zlog.String("version", "1.2.3"),
		zlog.Int("pid", os.Getpid()),
	)

	// Simulate application behavior
	app := &Application{
		env: env,
	}
	app.Run()

	zlog.Info("Application shutdown complete")
}

type Application struct {
	env string
}

func (a *Application) Run() {
	// Simulate various application activities

	// Debug information (only visible in development)
	zlog.Debug("Configuration loaded",
		zlog.String("config_path", "/etc/app/config.yaml"),
		zlog.String("database_host", "localhost:5432"),
		zlog.Int("worker_threads", 8),
	)

	// Simulate request handling
	for i := 0; i < 5; i++ {
		a.HandleRequest(fmt.Sprintf("req_%d", i))
		time.Sleep(100 * time.Millisecond)
	}

	// Simulate some errors
	a.SimulateErrors()

	// Emit some metrics
	a.EmitMetrics()

	// Security event (always logged)
	zlog.Emit(zlog.SECURITY, "Suspicious activity detected",
		zlog.String("ip", "192.168.1.100"),
		zlog.String("pattern", "sql_injection_attempt"),
		zlog.String("action", "blocked"),
	)
}

func (a *Application) HandleRequest(requestID string) {
	start := time.Now()

	// Debug log - only in development
	zlog.Debug("Request started",
		zlog.String("request_id", requestID),
		zlog.String("stage", "parsing"),
	)

	// Simulate processing
	processingTime := time.Duration(rand.Intn(100)+20) * time.Millisecond
	time.Sleep(processingTime)

	// More debug logs
	zlog.Debug("Database query executed",
		zlog.String("request_id", requestID),
		zlog.String("query", "SELECT * FROM users WHERE active = true"),
		zlog.Int("rows_returned", rand.Intn(50)),
	)

	duration := time.Since(start)

	// Info log - visible in both environments
	zlog.Info("Request completed",
		zlog.String("request_id", requestID),
		zlog.Duration("duration", duration),
		zlog.Int("status", 200),
	)

	// Emit metric
	zlog.Emit(zlog.METRIC, "http.request.duration",
		zlog.String("request_id", requestID),
		zlog.Float64("value", duration.Seconds()*1000),
		zlog.String("unit", "ms"),
	)
}

func (a *Application) SimulateErrors() {
	// Simulate different error scenarios

	// Warning - visible in both environments
	zlog.Warn("Cache miss rate high",
		zlog.Float64("miss_rate", 0.45),
		zlog.String("cache_name", "user_sessions"),
		zlog.String("recommendation", "increase_cache_size"),
	)

	// Error - always visible
	zlog.Error("Database connection failed",
		zlog.Err(fmt.Errorf("connection refused")),
		zlog.String("host", "db-replica-2"),
		zlog.Int("retry_count", 3),
		zlog.Bool("failover_initiated", true),
	)

	// Debug information about error handling
	zlog.Debug("Error recovery initiated",
		zlog.String("strategy", "circuit_breaker"),
		zlog.Int("cooldown_seconds", 30),
		zlog.String("fallback", "cache_only_mode"),
	)
}

func (a *Application) EmitMetrics() {
	// Various metrics that might be handled differently per environment

	metrics := []struct {
		name  string
		value float64
		unit  string
	}{
		{"memory.usage", 245.6, "MB"},
		{"cpu.usage", 65.2, "percent"},
		{"active.connections", 142, "count"},
		{"queue.length", 28, "messages"},
	}

	for _, m := range metrics {
		zlog.Emit(zlog.METRIC, m.name,
			zlog.Float64("value", m.value),
			zlog.String("unit", m.unit),
			zlog.String("host", "app-server-1"),
			zlog.String("environment", a.env),
		)
	}
}

// configureDevelopment sets up logging for development environment.
func configureDevelopment() {
	fmt.Println("🚀 Starting in DEVELOPMENT mode")

	// Everything goes to console for easy debugging
	zlog.EnableStandardLogging(os.Stderr)
	zlog.EnableDebugLogging(os.Stderr)

	// Also write to local files for persistence
	debugFile, _ := os.Create("debug.log")
	zlog.EnableDebugLogging(debugFile)

	// Metrics to a separate file for analysis
	metricsFile, _ := os.Create("metrics.log")
	zlog.EnableMetricLogging(metricsFile)

	// Security events to console AND file
	securityFile, _ := os.Create("security.log")
	securitySink := zlog.NewWriterSink(securityFile)
	zlog.RouteSignal(zlog.SECURITY, securitySink)
	zlog.RouteSignal(zlog.SECURITY, zlog.NewWriterSink(os.Stderr))

	fmt.Println("📝 Debug logs: console + debug.log")
	fmt.Println("📊 Metrics: metrics.log")
	fmt.Println("🔒 Security: console + security.log")
	fmt.Println("")
}

// configureProduction sets up logging for production environment.
func configureProduction() {
	// In production, we're more selective

	// Standard logs (INFO, WARN, ERROR) to stdout for container logs
	zlog.EnableStandardLogging(os.Stdout)

	// Audit logs to a separate sink (would be S3, CloudWatch, etc.)
	auditSink := NewCloudSink("audit-logs-bucket")
	zlog.RouteSignal(zlog.AUDIT, auditSink)
	zlog.RouteSignal(zlog.SECURITY, auditSink)

	// Metrics to a metrics aggregator (would be Prometheus, StatsD, etc.)
	metricsSink := NewMetricsAggregatorSink()
	zlog.RouteSignal(zlog.METRIC, metricsSink)

	// No DEBUG logs in production!
	// This significantly reduces log volume and improves performance

	// Could also add sampling for high-volume events
	// sampledSink := NewSampledSink(0.1, standardSink) // 10% sampling
}

// CloudSink simulates sending logs to cloud storage.
type CloudSink struct {
	bucket string
	buffer []zlog.Event
}

func NewCloudSink(bucket string) *CloudSink {
	return &CloudSink{
		bucket: bucket,
		buffer: make([]zlog.Event, 0, 100),
	}
}

func (s *CloudSink) Write(event zlog.Event) error {
	// In production, this would batch and send to S3, CloudWatch, etc.
	s.buffer = append(s.buffer, event)

	// Flush when buffer is full
	if len(s.buffer) >= 100 {
		fmt.Printf("[CloudSink] Flushing %d events to %s\n", len(s.buffer), s.bucket)
		s.buffer = s.buffer[:0]
	}

	return nil
}

func (s *CloudSink) Name() string {
	return fmt.Sprintf("cloud:%s", s.bucket)
}

// MetricsAggregatorSink simulates sending metrics to an aggregator.
type MetricsAggregatorSink struct {
	metrics map[string][]float64
}

func NewMetricsAggregatorSink() *MetricsAggregatorSink {
	return &MetricsAggregatorSink{
		metrics: make(map[string][]float64),
	}
}

func (s *MetricsAggregatorSink) Write(event zlog.Event) error {
	// In production, this would send to Prometheus, DataDog, etc.

	// Extract metric value
	for _, field := range event.Fields {
		if field.Key == "value" {
			if val, ok := field.Value.(float64); ok {
				s.metrics[event.Message] = append(s.metrics[event.Message], val)
			}
		}
	}

	// Periodically aggregate and send
	if rand.Float32() < 0.1 { // 10% chance to "flush"
		fmt.Printf("[MetricsAggregator] Sending %d metric types\n", len(s.metrics))
	}

	return nil
}

func (s *MetricsAggregatorSink) Name() string {
	return "metrics-aggregator"
}

// SampledSink wraps another sink and only forwards a percentage of events.
type SampledSink struct {
	sink zlog.Sink
	rate float64
}

func NewSampledSink(rate float64, sink zlog.Sink) *SampledSink {
	return &SampledSink{
		rate: rate,
		sink: sink,
	}
}

func (s *SampledSink) Write(event zlog.Event) error {
	if rand.Float64() < s.rate {
		return s.sink.Write(event)
	}
	return nil
}

func (s *SampledSink) Name() string {
	return fmt.Sprintf("sampled(%s)", s.sink.Name())
}

// DevSink adds colorful output for development.
type DevSink struct {
	writer io.Writer
}

func NewDevSink(w io.Writer) *DevSink {
	return &DevSink{writer: w}
}

func (s *DevSink) Write(event zlog.Event) error {
	// Add colors based on signal
	var color string
	switch event.Signal {
	case zlog.DEBUG:
		color = "\033[36m" // Cyan
	case zlog.INFO:
		color = "\033[32m" // Green
	case zlog.WARN:
		color = "\033[33m" // Yellow
	case zlog.ERROR, zlog.FATAL:
		color = "\033[31m" // Red
	default:
		color = "\033[0m" // Reset
	}

	fmt.Fprintf(s.writer, "%s[%s] %s%s\033[0m\n",
		color,
		event.Signal,
		event.Time.Format("15:04:05"),
		event.Message,
	)

	return nil
}

func (s *DevSink) Name() string {
	return "dev-console"
}

func init() {
	// Clean up any existing log files
	os.Remove("debug.log")
	os.Remove("metrics.log")
	os.Remove("security.log")

	// Seed random
	rand.Seed(time.Now().UnixNano())
}
