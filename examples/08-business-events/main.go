package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/zoobzio/zlog"
)

// Define custom business event signals.
const (
	REVENUE    = zlog.Signal("REVENUE")
	ENGAGEMENT = zlog.Signal("ENGAGEMENT")
	CONVERSION = zlog.Signal("CONVERSION")
	RETENTION  = zlog.Signal("RETENTION")
	EXPERIMENT = zlog.Signal("EXPERIMENT")
)

func main() {
	// Operational logs to stderr
	zlog.EnableStandardLogging(os.Stderr)

	// Business events to a separate file
	businessFile, err := os.Create("business-events.json")
	if err != nil {
		zlog.Fatal("Failed to create business events file", zlog.Err(err))
	}
	defer businessFile.Close()

	// Create a business events sink that listens to all business signals
	businessSink := zlog.NewWriterSink(businessFile)
	zlog.RouteSignal(REVENUE, businessSink)
	zlog.RouteSignal(ENGAGEMENT, businessSink)
	zlog.RouteSignal(CONVERSION, businessSink)
	zlog.RouteSignal(RETENTION, businessSink)
	zlog.RouteSignal(EXPERIMENT, businessSink)

	// Start the platform
	zlog.Info("E-commerce platform started")

	platform := &EcommercePlatform{
		users: []User{
			{ID: "user123", Name: "Alice", Plan: "free", SignupDate: time.Now().AddDate(0, -1, 0)},
			{ID: "user456", Name: "Bob", Plan: "trial", SignupDate: time.Now().AddDate(0, 0, -7)},
			{ID: "user789", Name: "Charlie", Plan: "trial", SignupDate: time.Now().AddDate(0, 0, -14)},
			{ID: "user101", Name: "David", Plan: "premium", SignupDate: time.Now().AddDate(0, -6, 0)},
			{ID: "user102", Name: "Eve", Plan: "professional", SignupDate: time.Now().AddDate(-1, 0, 0)},
		},
		products: []Product{
			{ID: "prod_001", Name: "Basic Widget", Price: 9.99, Category: "widgets"},
			{ID: "prod_002", Name: "Pro Widget", Price: 29.99, Category: "widgets"},
			{ID: "prod_003", Name: "Enterprise Suite", Price: 299.99, Category: "software"},
			{ID: "prod_004", Name: "Premium Plan", Price: 99.99, Category: "subscription"},
			{ID: "prod_005", Name: "Professional Plan", Price: 49.99, Category: "subscription"},
		},
	}

	// Simulate various business activities
	platform.SimulateBusinessActivity()

	zlog.Info("Business simulation complete")
}

type EcommercePlatform struct {
	users    []User
	products []Product
}

type User struct {
	SignupDate time.Time
	ID         string
	Name       string
	Plan       string
}

type Product struct {
	ID       string
	Name     string
	Category string
	Price    float64
}

func (p *EcommercePlatform) SimulateBusinessActivity() {
	// Simulate purchases
	p.SimulatePurchases()

	// Simulate feature engagement
	p.SimulateEngagement()

	// Simulate conversions
	p.SimulateConversions()

	// Simulate retention events
	p.SimulateRetention()

	// Simulate experiments
	p.SimulateExperiments()
}

func (p *EcommercePlatform) SimulatePurchases() {
	// Alice makes a purchase
	user := p.users[0]
	product := p.products[3] // Premium Plan

	zlog.Emit(REVENUE, "Purchase completed",
		zlog.String("user_id", user.ID),
		zlog.String("user_name", user.Name),
		zlog.Float64("amount", product.Price),
		zlog.String("currency", "USD"),
		zlog.String("product_id", product.ID),
		zlog.String("product_name", product.Name),
		zlog.String("category", product.Category),
		zlog.String("payment_method", "credit_card"),
		zlog.String("referrer", "email_campaign"),
	)

	// David renews subscription
	user = p.users[3]
	zlog.Emit(REVENUE, "Subscription renewed",
		zlog.String("user_id", user.ID),
		zlog.Float64("amount", 99.99),
		zlog.String("currency", "USD"),
		zlog.String("plan", "premium"),
		zlog.String("billing_period", "monthly"),
		zlog.Float64("mrr", 99.99),
		zlog.Int("renewal_count", 6),
	)

	// Failed payment
	zlog.Emit(REVENUE, "Payment failed",
		zlog.String("user_id", "user999"),
		zlog.Float64("amount", 49.99),
		zlog.String("reason", "insufficient_funds"),
		zlog.String("plan", "professional"),
		zlog.Bool("retry_scheduled", true),
	)
}

func (p *EcommercePlatform) SimulateEngagement() {
	features := []string{
		"dashboard", "advanced_search", "bulk_export",
		"api_access", "team_collaboration", "analytics",
	}

	// Simulate various feature usage
	for i := 0; i < 5; i++ {
		user := p.users[rand.Intn(len(p.users))]
		feature := features[rand.Intn(len(features))]
		duration := rand.Intn(300) + 10 // 10-310 seconds

		zlog.Emit(ENGAGEMENT, "Feature used",
			zlog.String("user_id", user.ID),
			zlog.String("feature", feature),
			zlog.Int("duration_seconds", duration),
			zlog.Bool("success", rand.Float32() > 0.1),
			zlog.String("user_plan", user.Plan),
			zlog.Int("items_processed", rand.Intn(100)),
		)
	}

	// Session tracking
	user := p.users[1]
	zlog.Emit(ENGAGEMENT, "Session completed",
		zlog.String("user_id", user.ID),
		zlog.Int("duration_minutes", 25),
		zlog.Int("pages_viewed", 12),
		zlog.Int("actions_taken", 5),
		zlog.String("exit_page", "/settings"),
		zlog.Bool("goal_completed", true),
	)
}

func (p *EcommercePlatform) SimulateConversions() {
	// Trial to paid conversion
	user := p.users[2] // Charlie on trial
	trialDays := int(time.Since(user.SignupDate).Hours() / 24)

	zlog.Emit(CONVERSION, "Trial to paid conversion",
		zlog.String("user_id", user.ID),
		zlog.Int("trial_days", trialDays),
		zlog.String("plan", "professional"),
		zlog.Float64("mrr", 49.99),
		zlog.String("conversion_source", "in_app_upgrade"),
		zlog.String("key_feature_used", "team_collaboration"),
	)

	// Funnel progression
	steps := []string{"landing_page", "signup_form", "email_verify", "onboarding", "first_action"}
	for i, step := range steps {
		dropoffRate := float64(i) * 0.15 // 15% dropoff per step

		zlog.Emit(CONVERSION, "Funnel step completed",
			zlog.String("funnel", "signup_flow"),
			zlog.String("step", step),
			zlog.Int("step_number", i+1),
			zlog.Int("users_entered", 1000),
			zlog.Int("users_completed", int(1000*(1-dropoffRate))),
			zlog.Float64("completion_rate", 1-dropoffRate),
			zlog.Int("avg_time_seconds", 30+i*20),
		)
	}

	// Upgrade event
	zlog.Emit(CONVERSION, "Plan upgraded",
		zlog.String("user_id", "user555"),
		zlog.String("from_plan", "basic"),
		zlog.String("to_plan", "professional"),
		zlog.Float64("mrr_increase", 40.00),
		zlog.String("trigger", "hit_usage_limit"),
		zlog.Int("days_on_previous_plan", 45),
	)
}

func (p *EcommercePlatform) SimulateRetention() {
	// User returns after absence
	zlog.Emit(RETENTION, "User reactivated",
		zlog.String("user_id", "user777"),
		zlog.Int("days_inactive", 30),
		zlog.String("reactivation_source", "email_campaign"),
		zlog.String("campaign_name", "win_back_30_days"),
		zlog.Bool("made_purchase", true),
	)

	// Churn risk detected
	user := p.users[4]
	zlog.Emit(RETENTION, "Churn risk detected",
		zlog.String("user_id", user.ID),
		zlog.Float64("churn_probability", 0.78),
		zlog.String("risk_factors", "low_engagement,support_tickets,failed_payment"),
		zlog.Int("days_since_last_login", 21),
		zlog.Int("support_tickets_30d", 3),
		zlog.String("recommended_action", "personal_outreach"),
	)

	// Successful retention
	zlog.Emit(RETENTION, "Retention campaign success",
		zlog.String("user_id", "user888"),
		zlog.String("campaign_type", "feature_education"),
		zlog.String("engaged_feature", "analytics"),
		zlog.Int("usage_increase_percent", 150),
		zlog.Bool("canceled_churn", true),
	)
}

func (p *EcommercePlatform) SimulateExperiments() {
	// A/B test results
	experiments := []struct {
		name    string
		variant string
		metric  string
		value   float64
	}{
		{"checkout_flow_v2", "control", "conversion_rate", 0.032},
		{"checkout_flow_v2", "variant_a", "conversion_rate", 0.041},
		{"pricing_page_test", "control", "click_through_rate", 0.12},
		{"pricing_page_test", "variant_b", "click_through_rate", 0.18},
	}

	for _, exp := range experiments {
		zlog.Emit(EXPERIMENT, "Experiment data",
			zlog.String("experiment_name", exp.name),
			zlog.String("variant", exp.variant),
			zlog.String("metric", exp.metric),
			zlog.Float64("value", exp.value),
			zlog.Int("sample_size", 1000+rand.Intn(2000)),
			zlog.Float64("confidence", 0.95),
			zlog.Bool("statistically_significant", exp.value > 0.15),
		)
	}

	// Feature flag event
	zlog.Emit(EXPERIMENT, "Feature flag evaluated",
		zlog.String("flag_name", "new_dashboard"),
		zlog.String("user_id", "user123"),
		zlog.Bool("flag_value", true),
		zlog.String("user_segment", "power_users"),
		zlog.Int("rollout_percentage", 25),
	)
}

// In a real implementation, you might create specialized sinks for business events.
type BusinessEventSink struct {
	writer io.Writer
}

func NewBusinessEventSink(w io.Writer) *BusinessEventSink {
	return &BusinessEventSink{
		writer: w,
	}
}

// Helper to calculate customer lifetime value.
func calculateCLTV(monthlyRevenue float64, avgRetentionMonths int) float64 {
	return monthlyRevenue * float64(avgRetentionMonths)
}

// Helper to generate event IDs.
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func init() {
	// Clean up from previous runs
	os.Remove("business-events.json")

	// Seed random
	rand.Seed(time.Now().UnixNano())
}
