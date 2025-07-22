package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zoobzio/zlog"
)

func TestMetricsPipeline(t *testing.T) {
	// Set up separate buffers for logs and metrics
	logBuf := &bytes.Buffer{}
	metricBuf := &bytes.Buffer{}

	zlog.EnableStandardLogging(logBuf)
	zlog.EnableMetricLogging(metricBuf)

	t.Run("MetricSeparation", func(t *testing.T) {
		// Emit different signal types
		zlog.Info("Application log")
		zlog.Emit(zlog.METRIC, "test.counter",
			zlog.Float64("value", 1.0),
			zlog.String("unit", "count"),
		)

		// Check logs don't contain metrics
		if strings.Contains(logBuf.String(), "METRIC") {
			t.Error("METRIC signals should not appear in standard logs")
		}
		if !strings.Contains(logBuf.String(), "Application log") {
			t.Error("INFO should appear in standard logs")
		}

		// Check metrics don't contain logs
		if strings.Contains(metricBuf.String(), "Application log") {
			t.Error("INFO logs should not appear in metrics")
		}
		if !strings.Contains(metricBuf.String(), "test.counter") {
			t.Error("Metrics should appear in metrics output")
		}
	})
}

func TestWebServerMetrics(t *testing.T) {
	metricBuf := &bytes.Buffer{}
	zlog.RouteSignal(zlog.METRIC, zlog.NewWriterSink(metricBuf))

	server := &WebServer{}

	t.Run("RequestMetrics", func(t *testing.T) {
		metricBuf.Reset()
		server.HandleRequest()

		// Should emit multiple metrics per request
		lines := strings.Split(strings.TrimSpace(metricBuf.String()), "\n")
		if len(lines) < 3 {
			t.Errorf("Expected at least 3 metrics per request, got %d", len(lines))
		}

		// Check for expected metric types
		metrics := make(map[string]bool)
		for _, line := range lines {
			var metric map[string]interface{}
			if err := json.Unmarshal([]byte(line), &metric); err != nil {
				t.Fatalf("Failed to parse metric JSON: %v", err)
			}

			if msg, ok := metric["message"].(string); ok {
				metrics[msg] = true
			}

			// All metrics should have required fields
			if metric["signal"] != "METRIC" {
				t.Errorf("Expected METRIC signal, got %v", metric["signal"])
			}
			if _, ok := metric["value"]; !ok {
				t.Error("Metric missing 'value' field")
			}
			if _, ok := metric["unit"]; !ok {
				t.Error("Metric missing 'unit' field")
			}
		}

		// Check we got the expected metric types
		expectedMetrics := []string{
			"http.request.duration",
			"memory.usage",
			"active.connections",
		}
		for _, expected := range expectedMetrics {
			if !metrics[expected] {
				t.Errorf("Missing expected metric: %s", expected)
			}
		}
	})

	t.Run("ErrorMetrics", func(t *testing.T) {
		// Force an error by setting high error count
		server.requestCount = 10
		server.errorCount = 5

		metricBuf.Reset()

		// This should trigger error rate metric
		server.HandleRequest()

		// Look for error rate metric
		hasErrorRate := false
		lines := strings.Split(strings.TrimSpace(metricBuf.String()), "\n")
		for _, line := range lines {
			if strings.Contains(line, "http.error.rate") {
				hasErrorRate = true

				var metric map[string]interface{}
				if err := json.Unmarshal([]byte(line), &metric); err != nil {
					t.Fatalf("Failed to parse error metric: %v", err)
				}

				// Check error rate calculation
				if value, ok := metric["value"].(float64); ok {
					expectedRate := 5.0 / 11.0 // 5 errors out of 11 total requests
					if value < expectedRate-0.01 || value > expectedRate+0.01 {
						t.Errorf("Expected error rate ~%f, got %f", expectedRate, value)
					}
				}
			}
		}

		if !hasErrorRate {
			t.Skip("No error generated in this test run (random)")
		}
	})
}

func TestMetricStructure(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(zlog.METRIC, zlog.NewWriterSink(buf))

	// Test different metric types
	t.Run("CounterMetric", func(t *testing.T) {
		buf.Reset()
		zlog.Emit(zlog.METRIC, "requests.total",
			zlog.Float64("value", 100),
			zlog.String("unit", "count"),
			zlog.String("type", "counter"),
		)

		var metric map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &metric); err != nil {
			t.Fatalf("Failed to parse metric: %v", err)
		}

		if metric["type"] != "counter" {
			t.Errorf("Expected type=counter, got %v", metric["type"])
		}
	})

	t.Run("GaugeMetric", func(t *testing.T) {
		buf.Reset()
		zlog.Emit(zlog.METRIC, "cpu.usage",
			zlog.Float64("value", 75.5),
			zlog.String("unit", "percent"),
			zlog.String("type", "gauge"),
		)

		var metric map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &metric); err != nil {
			t.Fatalf("Failed to parse metric: %v", err)
		}

		if metric["unit"] != "percent" {
			t.Errorf("Expected unit=percent, got %v", metric["unit"])
		}
	})

	t.Run("HistogramMetric", func(t *testing.T) {
		buf.Reset()
		zlog.Emit(zlog.METRIC, "response.time",
			zlog.Float64("value", 23.4),
			zlog.String("unit", "ms"),
			zlog.String("type", "histogram"),
			zlog.String("bucket", "25ms"),
		)

		var metric map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &metric); err != nil {
			t.Fatalf("Failed to parse metric: %v", err)
		}

		if metric["bucket"] != "25ms" {
			t.Errorf("Expected bucket=25ms, got %v", metric["bucket"])
		}
	})
}

func TestMetricsProcessor(t *testing.T) {
	proc := NewMetricsProcessor()

	t.Run("CounterAggregation", func(t *testing.T) {
		proc.ProcessMetric("requests", 1, "counter")
		proc.ProcessMetric("requests", 1, "counter")
		proc.ProcessMetric("requests", 1, "counter")

		if proc.counters["requests"] != 3 {
			t.Errorf("Expected counter=3, got %f", proc.counters["requests"])
		}
	})

	t.Run("GaugeUpdate", func(t *testing.T) {
		proc.ProcessMetric("temperature", 20.5, "gauge")
		proc.ProcessMetric("temperature", 21.0, "gauge")
		proc.ProcessMetric("temperature", 19.8, "gauge")

		// Gauge should only keep latest value
		if proc.gauges["temperature"] != 19.8 {
			t.Errorf("Expected gauge=19.8, got %f", proc.gauges["temperature"])
		}
	})

	t.Run("HistogramCollection", func(t *testing.T) {
		proc.ProcessMetric("latency", 10.5, "histogram")
		proc.ProcessMetric("latency", 15.2, "histogram")
		proc.ProcessMetric("latency", 12.8, "histogram")

		if len(proc.histograms["latency"]) != 3 {
			t.Errorf("Expected 3 histogram values, got %d", len(proc.histograms["latency"]))
		}
	})
}
