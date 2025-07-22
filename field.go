package zlog

import (
	"time"
)

// Field represents a typed key-value pair for structured logging.
type Field struct {
	Value any       `json:"value"`
	Key   string    `json:"key"`
	Type  FieldType `json:"type"`
}

// FieldType defines the type of a log field - string-based for adapter extensibility.
type FieldType string

const (
	// Core field types (only what zlog needs internally).
	StringType     FieldType = "string"
	IntType        FieldType = "int"
	Int64Type      FieldType = "int64"
	Float64Type    FieldType = "float64"
	BoolType       FieldType = "bool"
	ErrorType      FieldType = "error"
	DurationType   FieldType = "duration"
	TimeType       FieldType = "time"
	ByteStringType FieldType = "bytestring"
	DataType       FieldType = "data"
	StringsType    FieldType = "strings"
)

// Type-safe field constructors.
func String(key, value string) Field {
	return Field{Key: key, Type: StringType, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Type: IntType, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Type: Int64Type, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Type: Float64Type, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Type: BoolType, Value: value}
}

func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Type: ErrorType, Value: nil}
	}
	return Field{Key: "error", Type: ErrorType, Value: err.Error()}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Type: DurationType, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Type: TimeType, Value: value}
}

func ByteString(key string, value []byte) Field {
	return Field{Key: key, Type: ByteStringType, Value: string(value)}
}

func Strings(key string, value []string) Field {
	return Field{Key: key, Type: StringsType, Value: value}
}

// Data creates a field for complex data types.
func Data[T any](key string, value T) Field {
	return Field{Key: key, Type: DataType, Value: value}
}
