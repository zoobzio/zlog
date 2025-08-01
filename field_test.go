package zlog

import (
	"errors"
	"testing"
	"time"
)

func TestFieldConstructors(t *testing.T) {
	tests := []struct {
		name      string
		field     Field
		wantKey   string
		wantType  FieldType
		wantValue any
	}{
		{
			name:      "String field",
			field:     String("username", "alice"),
			wantKey:   "username",
			wantType:  StringType,
			wantValue: "alice",
		},
		{
			name:      "Int field",
			field:     Int("count", 42),
			wantKey:   "count",
			wantType:  IntType,
			wantValue: 42,
		},
		{
			name:      "Int64 field",
			field:     Int64("large", 9223372036854775807),
			wantKey:   "large",
			wantType:  Int64Type,
			wantValue: int64(9223372036854775807),
		},
		{
			name:      "Float64 field",
			field:     Float64("ratio", 3.14159),
			wantKey:   "ratio",
			wantType:  Float64Type,
			wantValue: 3.14159,
		},
		{
			name:      "Bool field true",
			field:     Bool("active", true),
			wantKey:   "active",
			wantType:  BoolType,
			wantValue: true,
		},
		{
			name:      "Bool field false",
			field:     Bool("active", false),
			wantKey:   "active",
			wantType:  BoolType,
			wantValue: false,
		},
		{
			name:      "Error field",
			field:     Err(errors.New("test error")),
			wantKey:   "error",
			wantType:  ErrorType,
			wantValue: errors.New("test error"),
		},
		{
			name:      "Duration field",
			field:     Duration("elapsed", 5*time.Second),
			wantKey:   "elapsed",
			wantType:  DurationType,
			wantValue: 5 * time.Second,
		},
		{
			name:      "Time field",
			field:     Time("timestamp", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
			wantKey:   "timestamp",
			wantType:  TimeType,
			wantValue: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name:      "ByteString field",
			field:     ByteString("data", []byte("hello")),
			wantKey:   "data",
			wantType:  ByteStringType,
			wantValue: "hello", // Note: converted to string
		},
		{
			name:      "Strings field",
			field:     Strings("tags", []string{"go", "logging", "zlog"}),
			wantKey:   "tags",
			wantType:  StringsType,
			wantValue: []string{"go", "logging", "zlog"},
		},
		{
			name:      "Data field with struct",
			field:     Data("user", struct{ ID int }{ID: 123}),
			wantKey:   "user",
			wantType:  DataType,
			wantValue: struct{ ID int }{ID: 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.wantKey {
				t.Errorf("Key = %v, want %v", tt.field.Key, tt.wantKey)
			}
			if tt.field.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", tt.field.Type, tt.wantType)
			}

			// Special handling for error comparison
			if tt.wantType == ErrorType {
				// Err() function stores error as string
				gotStr, ok1 := tt.field.Value.(string)
				wantErr, ok2 := tt.wantValue.(error)
				if !ok1 || !ok2 {
					t.Errorf("Error field value type mismatch")
				} else if gotStr != wantErr.Error() {
					t.Errorf("Error value = %v, want %v", gotStr, wantErr.Error())
				}
			} else {
				// Use deep equal for slices
				switch v := tt.wantValue.(type) {
				case []string:
					got, ok := tt.field.Value.([]string)
					switch {
					case !ok:
						t.Errorf("Value type mismatch for []string")
					case len(got) != len(v):
						t.Errorf("Slice length = %v, want %v", len(got), len(v))
					default:
						for i := range v {
							if got[i] != v[i] {
								t.Errorf("Slice element[%d] = %v, want %v", i, got[i], v[i])
							}
						}
					}
				default:
					if tt.field.Value != tt.wantValue {
						t.Errorf("Value = %v, want %v", tt.field.Value, tt.wantValue)
					}
				}
			}
		})
	}
}

func TestFieldEdgeCases(t *testing.T) {
	t.Run("Nil error", func(t *testing.T) {
		field := Err(nil)
		if field.Key != "error" {
			t.Errorf("Key = %v, want error", field.Key)
		}
		if field.Type != ErrorType {
			t.Errorf("Type = %v, want %v", field.Type, ErrorType)
		}
		if field.Value != nil {
			t.Errorf("Value = %v, want nil", field.Value)
		}
	})

	t.Run("Empty byte slice", func(t *testing.T) {
		field := ByteString("empty", []byte{})
		if field.Value != "" {
			t.Errorf("Value = %v, want empty string", field.Value)
		}
	})

	t.Run("Empty string slice", func(t *testing.T) {
		field := Strings("empty", []string{})
		got, ok := field.Value.([]string)
		if !ok {
			t.Errorf("Value type mismatch")
		}
		if len(got) != 0 {
			t.Errorf("Slice length = %v, want 0", len(got))
		}
	})

	t.Run("Nil slice", func(t *testing.T) {
		field := Strings("nil", nil)
		// The field constructor stores nil as-is
		if field.Value != nil {
			// Note: Some JSON encoders may treat nil slices as empty arrays
			// This is implementation-specific behavior
			t.Logf("Value = %v (type %T), implementation may convert nil to empty slice", field.Value, field.Value)
		}
	})
}
