package zlog

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestSinkWithSampling(t *testing.T) {
	tests := []struct {
		name        string
		rate        float64
		eventCount  int
		expectedMin int
		expectedMax int
		tolerance   float64
	}{
		{
			name:        "10% sampling",
			rate:        0.1,
			eventCount:  1000,
			expectedMin: 95, // Allow 5% variance
			expectedMax: 105,
		},
		{
			name:        "50% sampling",
			rate:        0.5,
			eventCount:  1000,
			expectedMin: 495,
			expectedMax: 505,
		},
		{
			name:        "1% sampling",
			rate:        0.01,
			eventCount:  10000,
			expectedMin: 95,
			expectedMax: 105,
		},
		{
			name:        "0% sampling",
			rate:        0.0,
			eventCount:  100,
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "100% sampling",
			rate:        1.0,
			eventCount:  100,
			expectedMin: 100,
			expectedMax: 100,
		},
		{
			name:        "Rate above 1.0 clamps to 100%",
			rate:        1.5,
			eventCount:  100,
			expectedMin: 100,
			expectedMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var processedCount int64

			// Create a sink that counts processed events
			countingSink := NewSink("counter", func(_ context.Context, _ Event) error {
				atomic.AddInt64(&processedCount, 1)
				return nil
			})

			// Apply sampling
			sampledSink := countingSink.WithSampling(tt.rate)

			// Process events
			ctx := context.Background()
			event := NewEvent(INFO, "test", nil)

			for i := 0; i < tt.eventCount; i++ {
				_, err := sampledSink.Process(ctx, event)
				if err != nil {
					t.Fatalf("Process failed: %v", err)
				}
			}

			// Check results
			processed := int(atomic.LoadInt64(&processedCount))
			if processed < tt.expectedMin || processed > tt.expectedMax {
				t.Errorf("Expected %d-%d events to be processed, got %d",
					tt.expectedMin, tt.expectedMax, processed)
			}
		})
	}
}

func TestSinkWithProbabilisticSampling(t *testing.T) {
	// Test probabilistic sampling with larger numbers for statistical validity
	tests := []struct {
		name        string
		rate        float64
		eventCount  int
		expectedMin int
		expectedMax int
	}{
		{
			name:        "25% probabilistic sampling",
			rate:        0.25,
			eventCount:  10000,
			expectedMin: 2300, // Allow ~10% variance for randomness
			expectedMax: 2700,
		},
		{
			name:        "75% probabilistic sampling",
			rate:        0.75,
			eventCount:  10000,
			expectedMin: 7200,
			expectedMax: 7800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var processedCount int64

			countingSink := NewSink("counter", func(_ context.Context, _ Event) error {
				atomic.AddInt64(&processedCount, 1)
				return nil
			})

			sampledSink := countingSink.WithProbabilisticSampling(tt.rate)

			ctx := context.Background()
			event := NewEvent(INFO, "test", nil)

			for i := 0; i < tt.eventCount; i++ {
				_, err := sampledSink.Process(ctx, event)
				if err != nil {
					t.Fatalf("Process failed: %v", err)
				}
			}

			processed := int(atomic.LoadInt64(&processedCount))
			if processed < tt.expectedMin || processed > tt.expectedMax {
				t.Errorf("Expected %d-%d events to be processed, got %d",
					tt.expectedMin, tt.expectedMax, processed)
			}
		})
	}
}

func TestSamplingDeterminism(t *testing.T) {
	// Test that deterministic sampling produces consistent results
	var count1, count2 int64

	sink1 := NewSink("counter1", func(_ context.Context, _ Event) error {
		atomic.AddInt64(&count1, 1)
		return nil
	}).WithSampling(0.2) // 20% sampling

	sink2 := NewSink("counter2", func(_ context.Context, _ Event) error {
		atomic.AddInt64(&count2, 1)
		return nil
	}).WithSampling(0.2) // Same 20% sampling

	ctx := context.Background()

	// Process same sequence through both sinks
	for i := 0; i < 100; i++ {
		event := NewEvent(INFO, "test", nil)
		_, _ = sink1.Process(ctx, event) //nolint:errcheck // Testing determinism, not error handling
		_, _ = sink2.Process(ctx, event) //nolint:errcheck // Testing determinism, not error handling
	}

	// Both should have processed exactly 20 events (every 5th)
	if count1 != 20 || count2 != 20 {
		t.Errorf("Expected exactly 20 events for deterministic 20%% sampling, got %d and %d",
			count1, count2)
	}
}

func TestSamplingWithOtherAdapters(t *testing.T) {
	// Test that sampling works correctly with other adapters
	var processedCount int64
	var errorCount int64

	// Create a sink that fails 50% of the time
	unreliableSink := NewSink("unreliable", func(_ context.Context, _ Event) error {
		count := atomic.AddInt64(&processedCount, 1)
		if count%2 == 0 {
			atomic.AddInt64(&errorCount, 1)
			return errors.New("test error")
		}
		return nil
	})

	// Apply sampling first, then retry
	// This ensures only sampled events get retried
	sampledSink := unreliableSink.
		WithSampling(0.1). // Only 10% of events
		WithRetry(3)       // Retry the sampled events

	ctx := context.Background()
	event := NewEvent(INFO, "test", nil)

	// Process 1000 events
	for i := 0; i < 1000; i++ {
		_, _ = sampledSink.Process(ctx, event) //nolint:errcheck // Testing sampling with retries, not error handling
	}

	// Should have attempted to process ~100 events (10% of 1000)
	// With retries, some events will be processed multiple times
	attempts := int(atomic.LoadInt64(&processedCount))
	if attempts < 100 || attempts > 400 { // Wide range due to retries
		t.Errorf("Expected 100-400 processing attempts, got %d", attempts)
	}
}

func BenchmarkSampling(b *testing.B) {
	sink := NewSink("bench", func(_ context.Context, _ Event) error {
		return nil
	})

	ctx := context.Background()
	event := NewEvent(INFO, "benchmark", nil)

	b.Run("NoSampling", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sink.Process(ctx, event) //nolint:errcheck // Benchmarking, not error handling
		}
	})

	b.Run("10PercentSampling", func(b *testing.B) {
		sampledSink := sink.WithSampling(0.1)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sampledSink.Process(ctx, event) //nolint:errcheck // Benchmarking, not error handling
		}
	})

	b.Run("1PercentSampling", func(b *testing.B) {
		sampledSink := sink.WithSampling(0.01)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sampledSink.Process(ctx, event) //nolint:errcheck // Benchmarking, not error handling
		}
	})
}
