package zlog

import (
	"time"
)

// Field represents a typed key-value pair for structured logging.
//
// Fields provide type-safe structured data that can be processed by sinks.
// Unlike using map[string]interface{}, fields preserve type information and
// are created with zero allocations using the provided constructors.
//
// Fields are immutable after creation and safe to share between goroutines.
type Field struct {
	// Value holds the actual data
	Value any `json:"value"`

	// Key identifies this field
	Key string `json:"key"`

	// Type indicates how to interpret Value
	Type FieldType `json:"type"`
}

// FieldType identifies how a Field's Value should be interpreted.
//
// Using strings instead of iota allows sinks to handle types without importing
// zlog, making the system more extensible. Custom sinks can define their own
// types if needed.
type FieldType string

// Standard field types cover common logging use cases.
// Sinks can use the Type field to handle values appropriately.
const (
	// StringType for string values.
	StringType FieldType = "string"

	// IntType for int values.
	IntType FieldType = "int"

	// Int64Type for int64 values.
	Int64Type FieldType = "int64"

	// Float64Type for float64 values.
	Float64Type FieldType = "float64"

	// BoolType for boolean values.
	BoolType FieldType = "bool"

	// ErrorType for error values (stored as strings).
	ErrorType FieldType = "error"

	// DurationType for time.Duration values.
	DurationType FieldType = "duration"

	// TimeType for time.Time values.
	TimeType FieldType = "time"

	// ByteStringType for []byte values (often base64 encoded).
	ByteStringType FieldType = "bytestring"

	// DataType for arbitrary structured data.
	DataType FieldType = "data"

	// StringsType for []string values.
	StringsType FieldType = "strings"
)

// String creates a string field.
//
//	zlog.String("user_id", "123")
//	zlog.String("method", request.Method)
func String(key, value string) Field {
	return Field{Key: key, Type: StringType, Value: value}
}

// Int creates an integer field.
//
//	zlog.Int("status_code", 200)
//	zlog.Int("retry_count", attempts)
func Int(key string, value int) Field {
	return Field{Key: key, Type: IntType, Value: value}
}

// Int64 creates a 64-bit integer field.
//
//	zlog.Int64("user_id", userID)
//	zlog.Int64("timestamp", time.Now().Unix())
func Int64(key string, value int64) Field {
	return Field{Key: key, Type: Int64Type, Value: value}
}

// Float64 creates a floating-point field.
//
//	zlog.Float64("temperature", 98.6)
//	zlog.Float64("response_time", 1.234)
func Float64(key string, value float64) Field {
	return Field{Key: key, Type: Float64Type, Value: value}
}

// Bool creates a boolean field.
//
//	zlog.Bool("success", true)
//	zlog.Bool("is_authenticated", user.IsAuthenticated())
func Bool(key string, value bool) Field {
	return Field{Key: key, Type: BoolType, Value: value}
}

// Err creates an error field with key "error".
//
// The error is stored as a string. If err is nil, the field value is nil.
//
//	zlog.Error("Failed to connect", zlog.Err(err))
//	zlog.Info("Retry succeeded", zlog.Err(lastErr))
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Type: ErrorType, Value: nil}
	}
	return Field{Key: "error", Type: ErrorType, Value: err.Error()}
}

// Duration creates a time duration field.
//
//	zlog.Duration("latency", time.Since(start))
//	zlog.Duration("timeout", 30*time.Second)
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Type: DurationType, Value: value}
}

// Time creates a time.Time field.
//
//	zlog.Time("created_at", user.CreatedAt)
//	zlog.Time("next_retry", time.Now().Add(backoff))
func Time(key string, value time.Time) Field {
	return Field{Key: key, Type: TimeType, Value: value}
}

// ByteString creates a field for binary data.
//
// The bytes are converted to a string for storage. Sinks typically
// encode this as base64 or hex when formatting.
//
//	zlog.ByteString("request_body", body)
//	zlog.ByteString("hash", sha256.Sum256(data))
func ByteString(key string, value []byte) Field {
	return Field{Key: key, Type: ByteStringType, Value: string(value)}
}

// Strings creates a field for string slices.
//
//	zlog.Strings("tags", []string{"api", "v2", "public"})
//	zlog.Strings("errors", validationErrors)
func Strings(key string, value []string) Field {
	return Field{Key: key, Type: StringsType, Value: value}
}

// Data creates a field for arbitrary structured data.
//
// Use this for complex types that don't fit the standard field types.
// The value is stored as-is - sinks are responsible for serialization.
//
//	zlog.Data("user", user)
//	zlog.Data("request_headers", req.Header)
//	zlog.Data("metrics", map[string]int{"hits": 42, "misses": 7})
func Data[T any](key string, value T) Field {
	return Field{Key: key, Type: DataType, Value: value}
}
