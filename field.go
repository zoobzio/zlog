package zlog

import (
	"time"
)

// ZlogField represents a typed key-value pair for structured logging
type ZlogField struct {
	Key   string        `json:"key"`
	Type  ZlogFieldType `json:"type"`
	Value any           `json:"value"`
}

// ZlogFieldType defines the type of a log field - string-based for adapter extensibility
type ZlogFieldType string

const (
	// Core field types (only what zlog needs internally)
	StringType     ZlogFieldType = "string"
	IntType        ZlogFieldType = "int"
	Int64Type      ZlogFieldType = "int64"
	Float64Type    ZlogFieldType = "float64"
	BoolType       ZlogFieldType = "bool"
	ErrorType      ZlogFieldType = "error"
	DurationType   ZlogFieldType = "duration"
	TimeType       ZlogFieldType = "time"
	ByteStringType ZlogFieldType = "bytestring"
	DataType       ZlogFieldType = "data"
	StringsType    ZlogFieldType = "strings"
)

// Type-safe field constructors
func String(key, value string) ZlogField {
	return ZlogField{Key: key, Type: StringType, Value: value}
}

func Int(key string, value int) ZlogField {
	return ZlogField{Key: key, Type: IntType, Value: value}
}

func Int64(key string, value int64) ZlogField {
	return ZlogField{Key: key, Type: Int64Type, Value: value}
}

func Float64(key string, value float64) ZlogField {
	return ZlogField{Key: key, Type: Float64Type, Value: value}
}

func Bool(key string, value bool) ZlogField {
	return ZlogField{Key: key, Type: BoolType, Value: value}
}

func Err(err error) ZlogField {
	return ZlogField{Key: "error", Type: ErrorType, Value: err}
}

func Duration(key string, value time.Duration) ZlogField {
	return ZlogField{Key: key, Type: DurationType, Value: value}
}

func Time(key string, value time.Time) ZlogField {
	return ZlogField{Key: key, Type: TimeType, Value: value}
}

func ByteString(key string, value []byte) ZlogField {
	return ZlogField{Key: key, Type: ByteStringType, Value: string(value)}
}

func Strings(key string, value []string) ZlogField {
	return ZlogField{Key: key, Type: StringsType, Value: value}
}

// Data creates a field for complex data types
func Data[T any](key string, value T) ZlogField {
	return ZlogField{Key: key, Type: DataType, Value: value}
}
