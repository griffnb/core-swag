package model

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/console"
)

type StructField struct {
	Name       string         `json:"name"`
	Type       types.Type     `json:"type"`
	TypeString string         `json:"type_string"` // For easier JSON serialization
	Tag        string         `json:"tag"`
	Fields     []*StructField `json:"fields"` // For nested structs
}

func (this *StructField) IsPublic() bool {
	_, ok := this.GetTags()["public"]
	return ok
}

func (this *StructField) GetTags() map[string]string {
	tags := strings.Split(this.Tag, " ")
	result := make(map[string]string)
	for _, tag := range tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			key := strings.Trim(parts[0], "`")
			value := strings.Trim(parts[1], "`")
			result[key] = strings.Trim(value, "\"")
		}
	}
	return result
}

// TypeEnumLookup is an interface for looking up enum values for a type
type TypeEnumLookup interface {
	GetEnumsForType(typeName string, file *ast.File) ([]EnumValue, error)
}

// EnumValue represents an enum constant
type EnumValue struct {
	Key     string
	Value   interface{}
	Comment string
}

// ToSpecSchema converts a StructField to OpenAPI spec.Schema
// propName: extracted from json tag (first part before comma)
// schema: the OpenAPI schema for this field
// required: true if omitempty is absent from json tag (or forceRequired is true)
// nestedTypes: list of struct type names encountered for recursive definition generation
// forceRequired: if true, field is always required regardless of omitempty tag
func (this *StructField) ToSpecSchema(
	public bool,
	forceRequired bool,
	enumLookup TypeEnumLookup,
) (propName string, schema *spec.Schema, required bool, nestedTypes []string, err error) {
	// Filter field if public mode and field is not public
	if public && !this.IsPublic() {
		return "", nil, false, nil, nil
	}

	// Check for swaggerignore tag
	tags := this.GetTags()
	if swaggerIgnore, ok := tags["swaggerignore"]; ok && strings.EqualFold(swaggerIgnore, "true") {
		console.Logger.Debug("$Red{$Bold{Ignoring field %s due to swaggerignore tag}}\n", this.Name)
		return "", nil, false, nil, nil
	}

	// Extract property name from json tag
	jsonTag := tags["json"]
	if jsonTag == "" {
		jsonTag = tags["column"]
	}
	if jsonTag == "" {
		// Skip fields without json or column tags - these are likely unexported
		// embedded fields that shouldn't be in the API schema
		return "", nil, false, nil, nil
	}

	parts := strings.Split(jsonTag, ",")
	propName = parts[0]

	// Check for omitempty to determine required
	// If forceRequired is true, field is always required
	if forceRequired {
		required = true
	} else {
		required = true
		for _, part := range parts[1:] {
			if strings.TrimSpace(part) == "omitempty" {
				required = false
				break
			}
		}
	}

	// Skip if json tag is "-"
	if propName == "-" {
		return "", nil, false, nil, nil
	}

	// Detect StructField[T] pattern and extract type parameter T
	typeStr := this.TypeString
	// Only use Type.String() if TypeString is empty or not set
	// This preserves manually set TypeString values (like "account.Properties")
	// instead of overriding with full path from Type.String()
	if this.Type != nil && this.TypeString == "" {
		typeStr = this.Type.String()
	}

	var extractedType string
	if strings.Contains(typeStr, "StructField[") || strings.Contains(typeStr, "IntConstantField[") || strings.Contains(typeStr, "StringField[") || strings.Contains(typeStr, "Field[") {
		// Extract type parameter using bracket parsing
		extractedType, err = extractGenericTypeParameter(typeStr)
		if err != nil {
			return "", nil, false, nil, fmt.Errorf("failed to extract type parameter from %s: %w", typeStr, err)
		}
	} else {
		extractedType = typeStr
	}

	// Build schema for the extracted type
	schema, nestedTypes, err = buildSchemaForType(extractedType, public, forceRequired, this.TypeString, enumLookup)
	if err != nil {
		return "", nil, false, nil, fmt.Errorf("failed to build schema for type %s: %w", extractedType, err)
	}

	return propName, schema, required, nestedTypes, nil
}

// normalizeTypeName converts a full module path type name to short form
// e.g., "github.com/griffnb/core-swag/testing/testdata/core_models/constants.UnionStatus" -> "constants.UnionStatus"
// Handles full paths, short names, and pointer types
func normalizeTypeName(typeStr string) string {
	// Remove pointer prefix
	isPointer := strings.HasPrefix(typeStr, "*")
	if isPointer {
		typeStr = strings.TrimPrefix(typeStr, "*")
	}

	// If it contains a slash, it's a full module path - extract package.Type
	if strings.Contains(typeStr, "/") {
		// Find the last two segments: package/TypeName
		lastSlash := strings.LastIndex(typeStr, "/")
		if lastSlash >= 0 {
			// Get everything after the last slash (e.g., "constants.UnionStatus")
			typeStr = typeStr[lastSlash+1:]
		}
	}

	if isPointer {
		return "*" + typeStr
	}
	return typeStr
}

// extractTypeParameter extracts the type parameter T from StructField[T]
// Handles nested brackets like StructField[map[string][]User]
func extractTypeParameter(typeStr string) (string, error) {
	return extractGenericTypeParameter(typeStr)
}

// extractGenericTypeParameter extracts the type parameter from any generic type
// Handles patterns like StructField[T], IntConstantField[T], StringField[T], etc.
// Also handles nested brackets like Field[map[string][]User]
func extractGenericTypeParameter(typeStr string) (string, error) {
	// Find the opening bracket
	idx := strings.Index(typeStr, "[")
	if idx == -1 {
		return "", fmt.Errorf("opening bracket [ not found in %s", typeStr)
	}

	// Start after "["
	start := idx + 1
	bracketCount := 1
	end := start

	// Count brackets to find matching closing bracket
	for end < len(typeStr) && bracketCount > 0 {
		switch typeStr[end] {
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}
		if bracketCount > 0 {
			end++
		}
	}

	if bracketCount != 0 {
		return "", fmt.Errorf("mismatched brackets in %s", typeStr)
	}

	extracted := typeStr[start:end]

	// Remove leading * if it's a pointer
	extracted = strings.TrimPrefix(extracted, "*")

	return extracted, nil
}

// buildSchemaForType builds an OpenAPI schema for a Go type string
// Returns schema, list of nested struct type names, and error
func buildSchemaForType(
	typeStr string,
	public bool,
	forceRequired bool,
	originalTypeStr string,
	enumLookup TypeEnumLookup,
) (*spec.Schema, []string, error) {
	var nestedTypes []string
	var debug bool
	if strings.Contains(typeStr, "constants.") {
		debug = true
	}
	if debug {
		console.Logger.Debug("Building schema for type: $Bold{%s} (original: $Bold{%s})\n", typeStr, originalTypeStr)
	}

	// Normalize type name to short form (package.Type instead of full/module/path/package.Type)
	typeStr = normalizeTypeName(typeStr)
	if debug && typeStr != originalTypeStr {
		console.Logger.Debug("Normalized type name to: $Bold{%s}\n", typeStr)
	}

	// Remove pointer prefix
	isPointer := strings.HasPrefix(typeStr, "*")
	if isPointer {
		typeStr = strings.TrimPrefix(typeStr, "*")
	}

	// Check if this is a fields wrapper type (StringField, IntField, etc.)
	// These should be treated as primitives, not struct types
	if isFieldsWrapperType(typeStr) {
		if debug {
			console.Logger.Debug("Detected fields wrapper type: $Bold{%s}\n", typeStr)
		}
		return getPrimitiveSchemaForFieldType(typeStr, originalTypeStr, enumLookup)
	}

	// Handle primitive types
	if isPrimitiveType(typeStr) {
		schema := primitiveTypeToSchema(typeStr)
		if debug {
			console.Logger.Debug("Detected Is Primitive type: $Bold{%s} Schema %+v\n", typeStr, schema)
		}
		return schema, nil, nil
	}

	// Handle arrays
	if strings.HasPrefix(typeStr, "[]") {
		elemType := strings.TrimPrefix(typeStr, "[]")
		elemSchema, elemNestedTypes, err := buildSchemaForType(elemType, public, forceRequired, originalTypeStr, enumLookup)
		if err != nil {
			return nil, nil, err
		}
		schema := spec.ArrayProperty(elemSchema)
		return schema, elemNestedTypes, nil
	}

	// Handle maps
	if strings.HasPrefix(typeStr, "map[") {
		// Extract value type
		bracketCount := 0
		valueStart := -1
		for i, ch := range typeStr {
			if ch == '[' {
				bracketCount++
			} else if ch == ']' {
				bracketCount--
				if bracketCount == 0 {
					valueStart = i + 1
					break
				}
			}
		}
		if valueStart == -1 {
			return nil, nil, fmt.Errorf("invalid map type: %s", typeStr)
		}
		valueType := typeStr[valueStart:]
		valueSchema, valueNestedTypes, err := buildSchemaForType(valueType, public, forceRequired, originalTypeStr, enumLookup)
		if err != nil {
			return nil, nil, err
		}
		schema := spec.MapProperty(valueSchema)
		return schema, valueNestedTypes, nil
	}

	// Handle struct types (including package-qualified names)
	// Filter out "any" and "interface{}" types - these should be treated as generic objects
	if typeStr == "any" || typeStr == "interface{}" {
		// Return a generic object schema, don't add to nestedTypes
		return &spec.Schema{}, nil, nil
	}

	// Check if this is an enum type - if so, inline the enum values instead of creating a reference
	if enumLookup != nil {
		if debug {
			console.Logger.Debug("Checking enum for type: $Bold{%s}\n", typeStr)
		}
		enums, err := enumLookup.GetEnumsForType(typeStr, nil)
		if err == nil && len(enums) > 0 {
			if debug {
				console.Logger.Debug("Detected Enum type: $Bold{%s} with %d values\n", typeStr, len(enums))
			}
			// This is an enum type - create an inline schema with enum values
			// Determine the base type from the first enum value
			schema := &spec.Schema{}
			if len(enums) > 0 {
				switch enums[0].Value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
					schema.Type = []string{"integer"}
				case string:
					schema.Type = []string{"string"}
				case float32, float64:
					schema.Type = []string{"number"}
				default:
					schema.Type = []string{"integer"} // default fallback
				}
			}
			applyEnumsToSchema(schema, enums)
			return schema, nil, nil
		}
		if debug {
			if err != nil {
				console.Logger.Debug("Error looking up enums for type: $Bold{%s}: $Red{%s}\n", typeStr, err.Error())
			}
		}
	} else {
		if debug {
			console.Logger.Debug("No enumLookup provided, skipping enum check for type: $Bold{%s}\n", typeStr)
		}
	}

	// Keep the full type name (including package prefix if present)
	// e.g., "account.Properties" should remain "account.Properties"
	typeName := typeStr

	// Validate that typeName doesn't have unbalanced brackets (malformed type names)
	// This can happen with complex generic types that weren't parsed correctly
	bracketDepth := 0
	for _, ch := range typeName {
		if ch == '[' {
			bracketDepth++
		} else if ch == ']' {
			bracketDepth--
		}
	}
	
	// If brackets are unbalanced, return a generic object schema instead of a bad reference
	if bracketDepth != 0 {
		console.Logger.Debug("Skipping reference creation for malformed type name with unbalanced brackets: %s\n", typeName)
		// Return a generic object schema
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}, nil, nil
	}

	// Add Public suffix if in public mode
	refName := typeName
	if public {
		refName = typeName + "Public"
	}

	// Create reference schema using the full type name
	schema := spec.RefSchema("#/definitions/" + refName)
	nestedTypes = append(nestedTypes, typeName)
	if debug {
		console.Logger.Debug("Created Ref Schema for type: $Bold{$Red{%s}} Ref: $Bold{#/definitions/%s}\n", typeStr, refName)
	}
	return schema, nestedTypes, nil
}

// isPrimitiveType checks if a type string is a Go primitive type
func isPrimitiveType(typeStr string) bool {
	primitives := map[string]bool{
		"string": true, "bool": true,
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"byte": true, "rune": true,
		"float32": true, "float64": true,
		"time.Time": true, "*time.Time": true,
		"decimal.Decimal": true, "*decimal.Decimal": true,
		"github.com/shopspring/decimal.Decimal": true, "*github.com/shopspring/decimal.Decimal": true,
		// UUID types
		"types.UUID": true, "*types.UUID": true,
		"uuid.UUID": true, "*uuid.UUID": true,
		"github.com/griffnb/core/lib/types.UUID": true, "*github.com/griffnb/core/lib/types.UUID": true,
		"github.com/google/uuid.UUID": true, "*github.com/google/uuid.UUID": true,
	}
	return primitives[typeStr]
}

// isFieldsWrapperType checks if a type is a fields package wrapper type
// like fields.StringField, fields.IntField, fields.StructField[T], etc.
func isFieldsWrapperType(typeStr string) bool {
	// Check for various field wrapper patterns
	return strings.Contains(typeStr, "fields.")
}

// getPrimitiveSchemaForFieldType returns the appropriate schema for a fields wrapper type
func getPrimitiveSchemaForFieldType(typeStr string, originalTypeStr string, enumLookup TypeEnumLookup) (*spec.Schema, []string, error) {
	// Check for IntConstantField and StringConstantField with enum type parameters
	if strings.Contains(typeStr, "fields.IntConstantField[") {
		// Extract enum type: fields.IntConstantField[constants.Role] -> constants.Role
		enumType := extractConstantFieldEnumType(originalTypeStr)
		if enumType != "" && enumLookup != nil {
			enums, err := enumLookup.GetEnumsForType(enumType, nil)
			if err == nil && len(enums) > 0 {
				schema := &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}
				applyEnumsToSchema(schema, enums)
				return schema, nil, nil
			}
		}
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.StringConstantField[") {
		// Extract enum type: fields.StringConstantField[constants.GlobalConfigKey] -> constants.GlobalConfigKey
		enumType := extractConstantFieldEnumType(originalTypeStr)
		if enumType != "" && enumLookup != nil {
			enums, err := enumLookup.GetEnumsForType(enumType, nil)
			if err == nil && len(enums) > 0 {
				schema := &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}
				applyEnumsToSchema(schema, enums)
				return schema, nil, nil
			}
		}
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.StringField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.IntField") || strings.Contains(typeStr, "fields.DecimalField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.UUIDField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "uuid"}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.BoolField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"boolean"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.FloatField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.TimeField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "date-time"}}, nil, nil
	}
	// Default to string for unknown field types
	return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
}

// extractConstantFieldEnumType extracts the enum type from IntConstantField[T] or StringConstantField[T]
func extractConstantFieldEnumType(typeStr string) string {
	// Look for pattern like "*fields.IntConstantField[constants.Role]" or "*fields.StringConstantField[constants.GlobalConfigKey]"
	if strings.Contains(typeStr, "ConstantField[") {
		start := strings.Index(typeStr, "[")
		end := strings.LastIndex(typeStr, "]")
		if start != -1 && end != -1 && end > start {
			return typeStr[start+1 : end]
		}
	}
	return ""
}

// applyEnumsToSchema applies enum values to a schema
func applyEnumsToSchema(schema *spec.Schema, enums []EnumValue) {
	if len(enums) == 0 {
		return
	}

	var enumValues []interface{}
	var varNames []string
	enumComments := make(map[string]string)
	var enumDescriptions []string

	dedupeMap := make(map[interface{}]bool)

	for _, enum := range enums {
		if _, exists := dedupeMap[enum.Value]; exists {
			continue
		}
		dedupeMap[enum.Value] = true
		enumValues = append(enumValues, enum.Value)
		varNames = append(varNames, enum.Key)
		enumDescriptions = append(enumDescriptions, enum.Comment)
		if enum.Comment != "" {
			enumComments[enum.Key] = enum.Comment
		}
	}

	schema.Enum = enumValues

	if schema.Extensions == nil {
		schema.Extensions = make(spec.Extensions)
	}
	schema.Extensions["x-enum-varnames"] = varNames

	if len(enumComments) > 0 {
		schema.Extensions["x-enum-comments"] = enumComments
		schema.Extensions["x-enum-descriptions"] = enumDescriptions
	}
}

// primitiveTypeToSchema converts a Go primitive type to OpenAPI schema
func primitiveTypeToSchema(typeStr string) *spec.Schema {
	switch typeStr {
	case "string":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}
	case "bool":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"boolean"}}}
	case "int", "uint":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}
	case "int8", "uint8", "int16", "uint16", "int32", "uint32", "byte", "rune":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}, Format: "int32"}}
	case "int64", "uint64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}, Format: "int64"}}
	case "float32":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}, Format: "float"}}
	case "float64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}, Format: "double"}}
	case "time.Time", "*time.Time":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "date-time"}}
	case "types.UUID", "*types.UUID", "uuid.UUID", "*uuid.UUID",
		"github.com/griffnb/core/lib/types.UUID", "*github.com/griffnb/core/lib/types.UUID",
		"github.com/google/uuid.UUID", "*github.com/google/uuid.UUID":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "uuid"}}
	case "decimal.Decimal", "*decimal.Decimal", "github.com/shopspring/decimal.Decimal", "*github.com/shopspring/decimal.Decimal":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}}}
	default:
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{typeStr}}}
	}
}
