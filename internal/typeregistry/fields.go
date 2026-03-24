package typeregistry

import (
	"strings"

	"github.com/go-openapi/spec"
)

// FieldsResult holds the resolution result for a fields package wrapper type.
type FieldsResult struct {
	// Schema is the resolved OpenAPI schema. Nil when InnerType is set
	// (caller must resolve the inner type themselves).
	Schema *spec.Schema

	// InnerType is the extracted generic type parameter (e.g., "constants.Role"
	// from "fields.IntConstantField[constants.Role]"). Empty for non-generic wrappers.
	InnerType string

	// IsEnum is true for IntConstantField[T] and StringConstantField[T].
	IsEnum bool

	// FallbackSchemaType is the schema type to use if enum lookup fails
	// (e.g., "integer" for IntConstantField, "string" for StringConstantField).
	// Only set when IsEnum is true.
	FallbackSchemaType string
}

// IsFieldsWrapper returns true if the type is a fields package wrapper type.
func IsFieldsWrapper(typeName string) bool {
	clean := strings.TrimPrefix(typeName, "*")
	return strings.HasPrefix(clean, "fields.") || strings.Contains(clean, "/fields.")
}

// ResolveFieldsWrapper resolves a fields package wrapper type to its OpenAPI schema.
// Returns false if the type is not a fields wrapper.
//
// For simple wrappers (StringField, IntField, etc.), returns a concrete Schema.
// For generic wrappers (IntConstantField[T], StructField[T]), returns InnerType
// with a nil Schema — the caller must resolve the inner type.
// For unknown fields wrappers, returns a default string schema.
func ResolveFieldsWrapper(typeName string) (*FieldsResult, bool) {
	if !IsFieldsWrapper(typeName) {
		return nil, false
	}

	// Check for constant field types with enum parameters first
	if strings.Contains(typeName, "IntConstantField[") {
		innerType := ExtractConstantFieldEnumType(typeName)
		return &FieldsResult{
			InnerType:          innerType,
			IsEnum:             true,
			FallbackSchemaType: "integer",
		}, true
	}
	if strings.Contains(typeName, "StringConstantField[") {
		innerType := ExtractConstantFieldEnumType(typeName)
		return &FieldsResult{
			InnerType:          innerType,
			IsEnum:             true,
			FallbackSchemaType: "string",
		}, true
	}

	// Check for StructField[T] — caller resolves inner type recursively
	if strings.Contains(typeName, "StructField[") {
		innerType := ExtractConstantFieldEnumType(typeName) // same bracket extraction
		return &FieldsResult{
			InnerType: innerType,
		}, true
	}

	// Simple wrapper types — return concrete schema
	if strings.Contains(typeName, "UUIDField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "uuid"}},
		}, true
	}
	if strings.Contains(typeName, "StringField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
		}, true
	}
	if strings.Contains(typeName, "IntField") || strings.Contains(typeName, "DecimalField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}},
		}, true
	}
	if strings.Contains(typeName, "BoolField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"boolean"}}},
		}, true
	}
	if strings.Contains(typeName, "FloatField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}}},
		}, true
	}
	if strings.Contains(typeName, "TimeField") {
		return &FieldsResult{
			Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "date-time"}},
		}, true
	}

	// Unknown fields wrapper — default to string
	return &FieldsResult{
		Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}},
	}, true
}

// ExtractConstantFieldEnumType extracts the type parameter from a generic fields type.
// e.g., "*fields.IntConstantField[constants.Role]" -> "constants.Role"
// Returns empty string if the type has no brackets.
func ExtractConstantFieldEnumType(typeStr string) string {
	if !strings.Contains(typeStr, "[") {
		return ""
	}
	start := strings.Index(typeStr, "[")
	end := strings.LastIndex(typeStr, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return typeStr[start+1 : end]
}
