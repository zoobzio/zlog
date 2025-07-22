package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/zoobzio/zlog"
)

func main() {
	// Standard logs to stderr
	zlog.EnableStandardLogging(os.Stderr)

	// Metrics to a separate file (in production, this would be a metrics system)
	metricsFile, err := os.Create("metrics.json")
	if err != nil {
		zlog.Fatal("Failed to create metrics file", zlog.Err(err))
	}
	defer metricsFile.Close()

	// Enable metrics collection
	zlog.EnableMetricLogging(metricsFile)

	// Start the application
	zlog.Info("Server started", zlog.Int("port", 8080))
	zlog.Info("Metrics collection enabled")

	// Simulate a web server
	server := &WebServer{}
	server.Run()
}

type WebServer struct {
	requestCount int
	errorCount   int
}

func (s *WebServer) Run() {
	// Simulate handling requests
	for i := 0; i < 10; i++ {
		s.HandleRequest()
		time.Sleep(100 * time.Millisecond)
	}

	// Report final stats
	zlog.Info("Server stats",
		zlog.Int("total_requests", s.requestCount),
		zlog.Int("total_errors", s.errorCount),
	)
}

func (s *WebServer) HandleRequest() {
	s.requestCount++

	// Simulate request processing with random latency
	start := time.Now()
	latency := time.Duration(rand.Intn(200)+50) * time.Millisecond
	time.Sleep(latency)
	duration := time.Since(start)

	// Simulate random status codes
	status := 200
	if rand.Float32() < 0.1 { // 10% error rate
		status = 500
		s.errorCount++
	}

	// Emit request metric
	zlog.Emit(zlog.METRIC, "http.request.duration",
		zlog.Float64("value", duration.Seconds()*1000), // Convert to ms
		zlog.String("unit", "ms"),
		zlog.String("method", "GET"),
		zlog.Int("status", status),
		zlog.String("endpoint", "/api/users"),
	)

	// Emit memory usage metric
	memoryMB := 40.0 + rand.Float64()*20.0 // Random between 40-60 MB
	zlog.Emit(zlog.METRIC, "memory.usage",
		zlog.Float64("value", memoryMB),
		zlog.String("unit", "MB"),
		zlog.String("type", "heap"),
	)

	// Emit active connections gauge
	activeConnections := 30 + rand.Intn(30) // Random between 30-60
	zlog.Emit(zlog.METRIC, "active.connections",
		zlog.Int("value", activeConnections),
		zlog.String("unit", "count"),
	)

	// Emit error rate if there was an error
	if status >= 500 {
		zlog.Emit(zlog.METRIC, "http.error.rate",
			zlog.Float64("value", float64(s.errorCount)/float64(s.requestCount)),
			zlog.String("unit", "ratio"),
			zlog.Int("status", status),
		)
	}
}

// MetricsProcessor shows how you might process metrics in a real system.
type MetricsProcessor struct {
	counters   map[string]float64
	gauges     map[string]float64
	histograms map[string][]float64
}

func NewMetricsProcessor() *MetricsProcessor {
	return &MetricsProcessor{
		counters:   make(map[string]float64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

// and process metrics based on their type.
func (p *MetricsProcessor) ProcessMetric(name string, value float64, metricType string) {
	switch metricType {
	case "counter":
		p.counters[name] += value
	case "gauge":
		p.gauges[name] = value
	case "histogram":
		p.histograms[name] = append(p.histograms[name], value)
	}
}

func init() {
	// Clean up from previous runs
	os.Remove("metrics.json")

	// Seed random for consistent examples
	rand.Seed(42)
}
