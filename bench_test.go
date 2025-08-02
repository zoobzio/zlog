package zlog

import (
	"context"
	"testing"
	"time"
)

// Benchmark-specific helpers

// noOpSink is a sink that does nothing - used to avoid stderr output during benchmarks.
var noOpSink = NewSink("benchmark-noop", func(_ context.Context, _ Log) error {
	return nil
})

// setupBenchmarks configures routing for benchmarks to avoid output.
func setupBenchmarks() {
	// Clear all routes to ensure no output during benchmarks
	defaultLogger = NewLogger[Fields]()
}

func init() {
	// Disable all logging output during benchmarks
	setupBenchmarks()
}

// BenchmarkEmit measures the core Emit function performance.
func BenchmarkEmit(b *testing.B) {
	// Route to no-op sink for benchmarks
	RouteSignal(INFO, noOpSink)
	defer setupBenchmarks() // Clean up after benchmark

	b.Run("NoFields", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Emit(INFO, "benchmark message")
		}
	})

	b.Run("WithFields", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Emit(INFO, "benchmark message",
				String("user", "test"),
				Int("count", 42),
				Bool("active", true))
		}
	})

	b.Run("ManyFields", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Emit(INFO, "benchmark message",
				String("user", "test"),
				Int("count", 42),
				Bool("active", true),
				Float64("score", 98.6),
				Duration("elapsed", time.Second),
				Time("timestamp", time.Now()),
				String("status", "active"),
				Int64("id", 123456),
				String("category", "benchmark"),
				Bool("verified", false))
		}
	})
}

// BenchmarkFields measures field creation performance.
func BenchmarkFields(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = String("key", "value")
		}
	})

	b.Run("Int", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Int("count", 42)
		}
	})

	b.Run("Mixed", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fields := []Field{
				String("user", "test"),
				Int("count", 42),
				Bool("active", true),
				Time("timestamp", time.Now()),
			}
			_ = fields
		}
	})
}

// BenchmarkConcurrent measures concurrent logging performance.
func BenchmarkConcurrent(b *testing.B) {
	RouteSignal(INFO, noOpSink)
	defer setupBenchmarks()

	b.Run("EmitParallel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Emit(INFO, "parallel message",
					String("goroutine", "test"))
			}
		})
	})
}

// BenchmarkSinks measures different sink implementations.
func BenchmarkSinks(b *testing.B) {
	ctx := context.Background()
	event := NewEvent(INFO, "benchmark message", []Field{
		String("user", "test"),
		Int("count", 42),
	})

	b.Run("NoOpSink", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := noOpSink.Process(ctx, event)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("JSONSink", func(b *testing.B) {
		// Create a JSON sink that simulates JSON serialization without I/O
		jsonSink := NewSink("json-bench", func(_ context.Context, e Log) error {
			// Simulate JSON serialization work
			data := make(map[string]interface{}, len(e.Data)+3)
			data["time"] = e.Time
			data["signal"] = e.Signal
			data["message"] = e.Message
			for _, f := range e.Data {
				data[f.Key] = f.Value
			}
			return nil
		})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := jsonSink.Process(ctx, event)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("FileSink", func(b *testing.B) {
		// Use temp file for benchmarking
		tempFile := b.TempDir() + "/bench.log"
		fileSink := NewRotatingFileSink(tempFile, 10*1024*1024, 3) // 10MB files
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := fileSink.Process(ctx, event)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HTTPSink", func(b *testing.B) {
		// Create HTTP sink that would send to a URL (but we won't actually use it)
		httpSink := NewHTTPSink("http://localhost:9999/logs")
		// For benchmarking, we just measure the serialization overhead
		// not actual network calls
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Measure the serialization overhead
			_, _ = httpSink.Process(ctx, event) //nolint:errcheck // Benchmarking HTTP sink setup overhead
		}
	})
}

// BenchmarkProduction measures realistic production scenarios.
func BenchmarkProduction(b *testing.B) {
	b.Run("TypicalLogging", func(b *testing.B) {
		// Typical production setup: JSON logging with some fields
		RouteSignal(INFO, noOpSink)
		RouteSignal(ERROR, noOpSink)
		defer setupBenchmarks()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Mix of different log levels and fields
			if i%10 == 0 {
				Emit(ERROR, "error occurred",
					String("error", "connection failed"),
					Int("retry", 3))
			} else {
				Emit(INFO, "request processed",
					String("user_id", "u123"),
					Int("status", 200),
					Duration("latency", 25*time.Millisecond))
			}
		}
	})

	b.Run("HighThroughput", func(b *testing.B) {
		// High throughput scenario with async sink
		asyncSink := noOpSink.WithAsync()
		RouteSignal(INFO, asyncSink)
		defer setupBenchmarks()

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Emit(INFO, "high throughput event",
					String("source", "benchmark"),
					Int64("timestamp", time.Now().UnixNano()))
			}
		})
	})

	b.Run("FilteredLogging", func(b *testing.B) {
		// Production scenario with filtering
		filteredSink := noOpSink.WithFilter(func(_ context.Context, e Log) bool {
			return e.Signal == ERROR || e.Signal == FATAL
		})
		RouteSignal(DEBUG, filteredSink)
		RouteSignal(INFO, filteredSink)
		RouteSignal(WARN, filteredSink)
		RouteSignal(ERROR, filteredSink)
		defer setupBenchmarks()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Most logs filtered out
			Emit(DEBUG, "debug message")
			Emit(INFO, "info message")
			if i%100 == 0 {
				Emit(ERROR, "error message") // Only these pass filter
			}
		}
	})
}

// BenchmarkRouting measures the routing and dispatch overhead.
func BenchmarkRouting(b *testing.B) {
	b.Run("SingleRoute", func(b *testing.B) {
		RouteSignal(INFO, noOpSink)
		defer setupBenchmarks()

		event := NewEvent(INFO, "test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			defaultLogger.Process(event)
		}
	})

	b.Run("MultipleRoutes", func(b *testing.B) {
		// Route same signal to multiple sinks
		RouteSignal(INFO, noOpSink)
		RouteSignal(INFO, noOpSink)
		RouteSignal(INFO, noOpSink)
		defer setupBenchmarks()

		event := NewEvent(INFO, "test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			defaultLogger.Process(event)
		}
	})
}
