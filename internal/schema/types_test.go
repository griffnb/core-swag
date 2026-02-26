package schema

import (
	"testing"

	"github.com/go-openapi/spec"
)

func TestIsSimplePrimitiveType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{"string is simple", "string", true},
		{"number is simple", "number", true},
		{"integer is simple", "integer", true},
		{"boolean is simple", "boolean", true},
		{"array is not simple", "array", false},
		{"object is not simple", "object", false},
		{"func is not simple", "func", false},
		{"custom type is not simple", "User", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := IsSimplePrimitiveType(tt.typeName)

			// Assert
			if got != tt.want {
				t.Errorf("IsSimplePrimitiveType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestIsPrimitiveType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{"string is primitive", "string", true},
		{"number is primitive", "number", true},
		{"integer is primitive", "integer", true},
		{"boolean is primitive", "boolean", true},
		{"array is primitive", "array", true},
		{"object is primitive", "object", true},
		{"func is primitive", "func", true},
		{"custom type is not primitive", "User", false},
		{"error is not primitive", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := IsPrimitiveType(tt.typeName)

			// Assert
			if got != tt.want {
				t.Errorf("IsPrimitiveType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestIsComplexSchema(t *testing.T) {
	t.Run("enum schema is complex", func(t *testing.T) {
		// Arrange
		schema := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"string"},
				Enum: []interface{}{"a", "b", "c"},
			},
		}

		// Act
		isComplex := IsComplexSchema(schema)

		// Assert
		if !isComplex {
			t.Error("expected enum schema to be complex")
		}
	})

	t.Run("deep array is complex", func(t *testing.T) {
		// Arrange
		schema := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"array", "array", "string"},
			},
		}

		// Act
		isComplex := IsComplexSchema(schema)

		// Assert
		if !isComplex {
			t.Error("expected deep array to be complex")
		}
	})

	t.Run("object type is complex", func(t *testing.T) {
		// Arrange
		schema := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
			},
		}

		// Act
		isComplex := IsComplexSchema(schema)

		// Assert
		if !isComplex {
			t.Error("expected object type to be complex")
		}
	})

	t.Run("simple string is not complex", func(t *testing.T) {
		// Arrange
		schema := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"string"},
			},
		}

		// Act
		isComplex := IsComplexSchema(schema)

		// Assert
		if isComplex {
			t.Error("expected simple string to not be complex")
		}
	})

	t.Run("simple array is not complex", func(t *testing.T) {
		// Arrange
		schema := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"array"},
			},
		}

		// Act
		isComplex := IsComplexSchema(schema)

		// Assert
		if isComplex {
			t.Error("expected simple array to not be complex")
		}
	})
}

func TestPrimitiveSchema(t *testing.T) {
	tests := []struct {
		name    string
		refType string
		want    []string
	}{
		{"creates string schema", "string", []string{"string"}},
		{"creates integer schema", "integer", []string{"integer"}},
		{"creates boolean schema", "boolean", []string{"boolean"}},
		{"creates number schema", "number", []string{"number"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			schema := PrimitiveSchema(tt.refType)

			// Assert
			if schema == nil {
				t.Fatal("expected schema to not be nil")
			}
			if len(schema.Type) != 1 {
				t.Fatalf("expected 1 type, got %d", len(schema.Type))
			}
			if schema.Type[0] != tt.want[0] {
				t.Errorf("expected type %s, got %s", tt.want[0], schema.Type[0])
			}
		})
	}
}

func TestBuildCustomSchema(t *testing.T) {
	t.Run("builds primitive schema", func(t *testing.T) {
		// Act
		schema, err := BuildCustomSchema([]string{"string"})
		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if schema == nil {
			t.Fatal("expected schema to not be nil")
		}
		if len(schema.Type) != 1 || schema.Type[0] != "string" {
			t.Error("expected string type")
		}
	})

	t.Run("builds array schema", func(t *testing.T) {
		// Act
		schema, err := BuildCustomSchema([]string{"array", "string"})
		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if schema == nil {
			t.Fatal("expected schema to not be nil")
		}
		if len(schema.Type) != 1 || schema.Type[0] != "array" {
			t.Error("expected array type")
		}
		if schema.Items == nil || schema.Items.Schema == nil {
			t.Fatal("expected items schema")
		}
		if len(schema.Items.Schema.Type) != 1 || schema.Items.Schema.Type[0] != "string" {
			t.Error("expected string items")
		}
	})

	t.Run("builds object schema", func(t *testing.T) {
		// Act
		schema, err := BuildCustomSchema([]string{"object", "string"})
		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if schema == nil {
			t.Fatal("expected schema to not be nil")
		}
		if len(schema.Type) != 1 || schema.Type[0] != "object" {
			t.Error("expected object type")
		}
		if schema.AdditionalProperties == nil || schema.AdditionalProperties.Schema == nil {
			t.Fatal("expected additional properties schema")
		}
	})

	t.Run("handles primitive keyword", func(t *testing.T) {
		// Act
		schema, err := BuildCustomSchema([]string{"primitive", "integer"})
		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if schema == nil {
			t.Fatal("expected schema to not be nil")
		}
		if len(schema.Type) != 1 || schema.Type[0] != "integer" {
			t.Error("expected integer type")
		}
	})

	t.Run("returns error for empty types", func(t *testing.T) {
		// Act
		schema, err := BuildCustomSchema([]string{})
		// Assert
		if err != nil {
			t.Errorf("expected no error for empty types, got %v", err)
		}
		if schema != nil {
			t.Error("expected nil schema for empty types")
		}
	})

	t.Run("returns error for array without item type", func(t *testing.T) {
		// Act
		_, err := BuildCustomSchema([]string{"array"})

		// Assert
		if err == nil {
			t.Error("expected error for array without item type")
		}
	})

	t.Run("returns error for primitive without type", func(t *testing.T) {
		// Act
		_, err := BuildCustomSchema([]string{"primitive"})

		// Assert
		if err == nil {
			t.Error("expected error for primitive without type")
		}
	})

	t.Run("returns error for invalid type", func(t *testing.T) {
		// Act
		_, err := BuildCustomSchema([]string{"invalid"})

		// Assert
		if err == nil {
			t.Error("expected error for invalid type")
		}
	})
}

func TestMergeSchema(t *testing.T) {
	t.Run("merges type", func(t *testing.T) {
		// Arrange
		dst := &spec.Schema{}
		src := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"string"},
			},
		}

		// Act
		result := MergeSchema(dst, src)

		// Assert
		if len(result.Type) != 1 || result.Type[0] != "string" {
			t.Error("expected type to be merged")
		}
	})

	t.Run("merges properties", func(t *testing.T) {
		// Arrange
		dst := &spec.Schema{}
		src := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Properties: map[string]spec.Schema{
					"name": {SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
				},
			},
		}

		// Act
		result := MergeSchema(dst, src)

		// Assert
		if len(result.Properties) != 1 {
			t.Error("expected properties to be merged")
		}
	})

	t.Run("merges description", func(t *testing.T) {
		// Arrange
		dst := &spec.Schema{}
		src := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "test description",
			},
		}

		// Act
		result := MergeSchema(dst, src)

		// Assert
		if result.Description != "test description" {
			t.Error("expected description to be merged")
		}
	})

	t.Run("merges nullable", func(t *testing.T) {
		// Arrange
		dst := &spec.Schema{}
		src := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Nullable: true,
			},
		}

		// Act
		result := MergeSchema(dst, src)

		// Assert
		if !result.Nullable {
			t.Error("expected nullable to be merged")
		}
	})

	t.Run("merges format", func(t *testing.T) {
		// Arrange
		dst := &spec.Schema{}
		src := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Format: "date-time",
			},
		}

		// Act
		result := MergeSchema(dst, src)

		// Assert
		if result.Format != "date-time" {
			t.Error("expected format to be merged")
		}
	})

	t.Run("does not overwrite with empty values", func(t *testing.T) {
		// Arrange
		dst := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:        []string{"string"},
				Description: "original",
			},
		}
		src := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				// Empty type should not overwrite
			},
		}

		// Act
		result := MergeSchema(dst, src)

		// Assert
		if len(result.Type) != 1 || result.Type[0] != "string" {
			t.Error("expected type to remain unchanged")
		}
		if result.Description != "original" {
			t.Error("expected description to remain unchanged")
		}
	})
}
