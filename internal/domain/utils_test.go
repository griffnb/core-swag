package domain

import (
	"testing"
)

func TestIsExtendedPrimitiveType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		// Basic Go primitives
		{"string", "string", true},
		{"int", "int", true},
		{"bool", "bool", true},
		{"float64", "float64", true},

		// Extended primitives
		{"time.Time", "time.Time", true},
		{"*time.Time", "*time.Time", true},
		{"uuid.UUID", "uuid.UUID", true},
		{"*uuid.UUID", "*uuid.UUID", true},
		{"types.UUID", "types.UUID", true},
		{"github.com/google/uuid.UUID", "github.com/google/uuid.UUID", true},
		{"decimal.Decimal", "decimal.Decimal", true},
		{"*decimal.Decimal", "*decimal.Decimal", true},
		{"github.com/shopspring/decimal.Decimal", "github.com/shopspring/decimal.Decimal", true},

		// Not primitives
		{"model.User", "model.User", false},
		{"User", "User", false},
		{"github.com/myapp/model.Account", "github.com/myapp/model.Account", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExtendedPrimitiveType(tt.typeName); got != tt.want {
				t.Errorf("IsExtendedPrimitiveType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestTransToValidPrimitiveSchema(t *testing.T) {
	tests := []struct {
		name         string
		typeName     string
		wantType     string
		wantFormat   string
	}{
		// Basic types
		{"string", "string", "string", ""},
		{"int", "int", "integer", ""},
		{"int32", "int32", "integer", "int32"},
		{"int64", "int64", "integer", "int64"},
		{"float32", "float32", "number", "float"},
		{"float64", "float64", "number", "double"},
		{"bool", "bool", "boolean", ""},

		// Extended primitives
		{"time.Time", "time.Time", "string", "date-time"},
		{"*time.Time", "*time.Time", "string", "date-time"},
		{"uuid.UUID", "uuid.UUID", "string", "uuid"},
		{"*uuid.UUID", "*uuid.UUID", "string", "uuid"},
		{"types.UUID", "types.UUID", "string", "uuid"},
		{"github.com/google/uuid.UUID", "github.com/google/uuid.UUID", "string", "uuid"},
		{"decimal.Decimal", "decimal.Decimal", "number", ""},
		{"*decimal.Decimal", "*decimal.Decimal", "number", ""},
		{"github.com/shopspring/decimal.Decimal", "github.com/shopspring/decimal.Decimal", "number", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := TransToValidPrimitiveSchema(tt.typeName)
			if schema == nil {
				t.Fatalf("TransToValidPrimitiveSchema(%q) returned nil", tt.typeName)
			}

			if len(schema.Type) == 0 {
				t.Errorf("TransToValidPrimitiveSchema(%q) has no type", tt.typeName)
				return
			}

			gotType := schema.Type[0]
			if gotType != tt.wantType {
				t.Errorf("TransToValidPrimitiveSchema(%q).Type = %q, want %q", tt.typeName, gotType, tt.wantType)
			}

			if schema.Format != tt.wantFormat {
				t.Errorf("TransToValidPrimitiveSchema(%q).Format = %q, want %q", tt.typeName, schema.Format, tt.wantFormat)
			}
		})
	}
}

func TestTransToValidPrimitiveSchema_TimeFormat(t *testing.T) {
	schema := TransToValidPrimitiveSchema("time.Time")

	if len(schema.Type) == 0 || schema.Type[0] != "string" {
		t.Errorf("time.Time should be type string, got %v", schema.Type)
	}

	if schema.Format != "date-time" {
		t.Errorf("time.Time should have format date-time, got %s", schema.Format)
	}
}

func TestTransToValidPrimitiveSchema_UUIDFormat(t *testing.T) {
	uuidTypes := []string{
		"uuid.UUID",
		"types.UUID",
		"github.com/google/uuid.UUID",
		"github.com/griffnb/core/lib/types.UUID",
	}

	for _, typeName := range uuidTypes {
		t.Run(typeName, func(t *testing.T) {
			schema := TransToValidPrimitiveSchema(typeName)

			if len(schema.Type) == 0 || schema.Type[0] != "string" {
				t.Errorf("%s should be type string, got %v", typeName, schema.Type)
			}

			if schema.Format != "uuid" {
				t.Errorf("%s should have format uuid, got %s", typeName, schema.Format)
			}
		})
	}
}

func TestTransToValidPrimitiveSchema_DecimalFormat(t *testing.T) {
	decimalTypes := []string{
		"decimal.Decimal",
		"github.com/shopspring/decimal.Decimal",
	}

	for _, typeName := range decimalTypes {
		t.Run(typeName, func(t *testing.T) {
			schema := TransToValidPrimitiveSchema(typeName)

			if len(schema.Type) == 0 || schema.Type[0] != "number" {
				t.Errorf("%s should be type number, got %v", typeName, schema.Type)
			}
		})
	}
}

func TestTransToValidPrimitiveSchema_PointerTypes(t *testing.T) {
	tests := []struct {
		typeName   string
		wantType   string
		wantFormat string
	}{
		{"*time.Time", "string", "date-time"},
		{"*uuid.UUID", "string", "uuid"},
		{"*decimal.Decimal", "number", ""},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			schema := TransToValidPrimitiveSchema(tt.typeName)

			if len(schema.Type) == 0 {
				t.Fatalf("%s returned schema with no type", tt.typeName)
			}

			if schema.Type[0] != tt.wantType {
				t.Errorf("%s should be type %s, got %s", tt.typeName, tt.wantType, schema.Type[0])
			}

			if schema.Format != tt.wantFormat {
				t.Errorf("%s should have format %s, got %s", tt.typeName, tt.wantFormat, schema.Format)
			}
		})
	}
}
