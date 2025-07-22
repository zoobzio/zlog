package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zoobzio/zlog"
)

func TestBusinessEventSignals(t *testing.T) {
	// Verify all business signals are unique
	signals := []zlog.Signal{REVENUE, ENGAGEMENT, CONVERSION, RETENTION, EXPERIMENT}
	seen := make(map[zlog.Signal]bool)

	for _, sig := range signals {
		if seen[sig] {
			t.Errorf("Duplicate signal: %s", sig)
		}
		seen[sig] = true
	}

	// Verify signal names
	expectedNames := map[zlog.Signal]string{
		REVENUE:    "REVENUE",
		ENGAGEMENT: "ENGAGEMENT",
		CONVERSION: "CONVERSION",
		RETENTION:  "RETENTION",
		EXPERIMENT: "EXPERIMENT",
	}

	for sig, expected := range expectedNames {
		if string(sig) != expected {
			t.Errorf("Signal %s has wrong name: %s", expected, sig)
		}
	}
}

func TestBusinessEventSeparation(t *testing.T) {
	logBuf := &bytes.Buffer{}
	bizBuf := &bytes.Buffer{}

	zlog.EnableStandardLogging(logBuf)

	// Route all business signals to business buffer
	bizSink := zlog.NewWriterSink(bizBuf)
	zlog.RouteSignal(REVENUE, bizSink)
	zlog.RouteSignal(ENGAGEMENT, bizSink)
	zlog.RouteSignal(CONVERSION, bizSink)

	t.Run("SignalIsolation", func(t *testing.T) {
		logBuf.Reset()
		bizBuf.Reset()

		// Operational log
		zlog.Info("Server started")

		// Business events
		zlog.Emit(REVENUE, "Test purchase", zlog.Float64("amount", 99.99))
		zlog.Emit(ENGAGEMENT, "Test engagement", zlog.String("feature", "test"))
		zlog.Emit(CONVERSION, "Test conversion", zlog.String("type", "trial"))

		// Check operational logs don't have business events
		logStr := logBuf.String()
		if strings.Contains(logStr, "REVENUE") ||
			strings.Contains(logStr, "ENGAGEMENT") ||
			strings.Contains(logStr, "CONVERSION") {
			t.Error("Business events should not appear in operational logs")
		}

		// Check business events don't have operational logs
		bizStr := bizBuf.String()
		if strings.Contains(bizStr, "Server started") {
			t.Error("Operational logs should not appear in business events")
		}

		// Verify all business events are present
		if !strings.Contains(bizStr, "Test purchase") {
			t.Error("REVENUE event missing from business events")
		}
		if !strings.Contains(bizStr, "Test engagement") {
			t.Error("ENGAGEMENT event missing from business events")
		}
		if !strings.Contains(bizStr, "Test conversion") {
			t.Error("CONVERSION event missing from business events")
		}
	})
}

func TestRevenueEvents(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(REVENUE, zlog.NewWriterSink(buf))

	// platform variable removed - not needed for these tests

	t.Run("PurchaseEvent", func(t *testing.T) {
		buf.Reset()

		// Emit a purchase event
		zlog.Emit(REVENUE, "Purchase completed",
			zlog.String("user_id", "test123"),
			zlog.Float64("amount", 99.99),
			zlog.String("currency", "USD"),
			zlog.String("product_id", "prod_001"),
		)

		var event map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse revenue event: %v", err)
		}

		// Verify event structure
		if event["signal"] != "REVENUE" {
			t.Errorf("Expected signal=REVENUE, got %v", event["signal"])
		}
		if event["amount"] != 99.99 {
			t.Errorf("Expected amount=99.99, got %v", event["amount"])
		}
		if event["currency"] != "USD" {
			t.Errorf("Expected currency=USD, got %v", event["currency"])
		}
	})

	t.Run("SubscriptionMetrics", func(t *testing.T) {
		buf.Reset()

		// Test MRR tracking
		zlog.Emit(REVENUE, "Subscription renewed",
			zlog.String("user_id", "test456"),
			zlog.Float64("mrr", 49.99),
			zlog.String("billing_period", "monthly"),
			zlog.Int("renewal_count", 3),
		)

		var event map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse subscription event: %v", err)
		}

		if event["mrr"] != 49.99 {
			t.Errorf("Expected mrr=49.99, got %v", event["mrr"])
		}
		if event["renewal_count"] != float64(3) { // JSON numbers are float64
			t.Errorf("Expected renewal_count=3, got %v", event["renewal_count"])
		}
	})
}

func TestEngagementEvents(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(ENGAGEMENT, zlog.NewWriterSink(buf))

	t.Run("FeatureUsage", func(t *testing.T) {
		buf.Reset()

		zlog.Emit(ENGAGEMENT, "Feature used",
			zlog.String("user_id", "test_user"),
			zlog.String("feature", "bulk_export"),
			zlog.Int("duration_seconds", 45),
			zlog.Bool("success", true),
		)

		var event map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse engagement event: %v", err)
		}

		if event["feature"] != "bulk_export" {
			t.Errorf("Expected feature=bulk_export, got %v", event["feature"])
		}
		if event["success"] != true {
			t.Errorf("Expected success=true, got %v", event["success"])
		}
	})

	t.Run("SessionTracking", func(t *testing.T) {
		buf.Reset()

		zlog.Emit(ENGAGEMENT, "Session completed",
			zlog.String("user_id", "test_user"),
			zlog.Int("duration_minutes", 30),
			zlog.Int("pages_viewed", 15),
			zlog.Bool("goal_completed", true),
		)

		events := parseBusinessEvents(t, buf.String())
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		event := events[0]
		if event["duration_minutes"] != float64(30) {
			t.Errorf("Expected duration_minutes=30, got %v", event["duration_minutes"])
		}
	})
}

func TestConversionEvents(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(CONVERSION, zlog.NewWriterSink(buf))

	t.Run("TrialConversion", func(t *testing.T) {
		buf.Reset()

		zlog.Emit(CONVERSION, "Trial to paid conversion",
			zlog.String("user_id", "trial_user"),
			zlog.Int("trial_days", 14),
			zlog.String("plan", "professional"),
			zlog.Float64("mrr", 49.99),
		)

		var event map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse conversion event: %v", err)
		}

		if event["trial_days"] != float64(14) {
			t.Errorf("Expected trial_days=14, got %v", event["trial_days"])
		}
		if event["plan"] != "professional" {
			t.Errorf("Expected plan=professional, got %v", event["plan"])
		}
	})

	t.Run("FunnelTracking", func(t *testing.T) {
		buf.Reset()

		// Track funnel progression
		steps := []string{"landing", "signup", "activate"}
		for i, step := range steps {
			zlog.Emit(CONVERSION, "Funnel step",
				zlog.String("funnel", "onboarding"),
				zlog.String("step", step),
				zlog.Int("step_number", i+1),
				zlog.Float64("completion_rate", 1.0-float64(i)*0.2),
			)
		}

		events := parseBusinessEvents(t, buf.String())
		if len(events) != 3 {
			t.Errorf("Expected 3 funnel events, got %d", len(events))
		}

		// Verify funnel steps are tracked correctly
		for i, event := range events {
			if event["step_number"] != float64(i+1) {
				t.Errorf("Expected step_number=%d, got %v", i+1, event["step_number"])
			}
		}
	})
}

func TestRetentionEvents(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(RETENTION, zlog.NewWriterSink(buf))

	t.Run("ChurnRisk", func(t *testing.T) {
		buf.Reset()

		zlog.Emit(RETENTION, "Churn risk detected",
			zlog.String("user_id", "at_risk_user"),
			zlog.Float64("churn_probability", 0.85),
			zlog.String("risk_factors", "low_engagement,no_recent_login"),
			zlog.Int("days_since_last_login", 30),
		)

		var event map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse retention event: %v", err)
		}

		if prob := event["churn_probability"].(float64); prob != 0.85 {
			t.Errorf("Expected churn_probability=0.85, got %v", prob)
		}
		if days := event["days_since_last_login"].(float64); days != 30 {
			t.Errorf("Expected days_since_last_login=30, got %v", days)
		}
	})
}

func TestExperimentEvents(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(EXPERIMENT, zlog.NewWriterSink(buf))

	t.Run("ABTestResults", func(t *testing.T) {
		buf.Reset()

		zlog.Emit(EXPERIMENT, "Experiment data",
			zlog.String("experiment_name", "checkout_redesign"),
			zlog.String("variant", "variant_a"),
			zlog.String("metric", "conversion_rate"),
			zlog.Float64("value", 0.045),
			zlog.Int("sample_size", 1500),
			zlog.Bool("statistically_significant", true),
		)

		var event map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse experiment event: %v", err)
		}

		if event["experiment_name"] != "checkout_redesign" {
			t.Errorf("Expected experiment_name=checkout_redesign, got %v", event["experiment_name"])
		}
		if event["statistically_significant"] != true {
			t.Error("Expected statistically_significant=true")
		}
	})

	t.Run("FeatureFlags", func(t *testing.T) {
		buf.Reset()

		zlog.Emit(EXPERIMENT, "Feature flag evaluated",
			zlog.String("flag_name", "new_ui"),
			zlog.String("user_id", "test_user"),
			zlog.Bool("flag_value", true),
			zlog.Int("rollout_percentage", 50),
		)

		var event map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse feature flag event: %v", err)
		}

		if event["rollout_percentage"] != float64(50) {
			t.Errorf("Expected rollout_percentage=50, got %v", event["rollout_percentage"])
		}
	})
}

func TestCLTVCalculation(t *testing.T) {
	tests := []struct {
		monthlyRevenue     float64
		avgRetentionMonths int
		expectedCLTV       float64
	}{
		{50.00, 12, 600.00},
		{99.99, 24, 2399.76},
		{29.99, 6, 179.94},
	}

	for _, tt := range tests {
		cltv := calculateCLTV(tt.monthlyRevenue, tt.avgRetentionMonths)
		// Allow small floating point differences
		diff := cltv - tt.expectedCLTV
		if diff < -0.01 || diff > 0.01 {
			t.Errorf("CLTV calculation wrong: got %v, want %v", cltv, tt.expectedCLTV)
		}
	}
}

func TestEventIDGeneration(t *testing.T) {
	// Test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateEventID()
		if ids[id] {
			t.Errorf("Duplicate event ID: %s", id)
		}
		ids[id] = true

		// Verify format
		if !strings.HasPrefix(id, "evt_") {
			t.Errorf("Event ID should start with 'evt_', got: %s", id)
		}

		// Small delay to ensure different timestamps
		time.Sleep(time.Microsecond)
	}
}

// Helper to parse multiple business events.
func parseBusinessEvents(t *testing.T, output string) []map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	events := make([]map[string]interface{}, 0)

	for _, line := range lines {
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("Failed to parse business event: %v\nLine: %s", err, line)
		}
		events = append(events, event)
	}

	return events
}
