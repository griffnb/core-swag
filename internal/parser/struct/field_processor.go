package structparser

import (
	"go/ast"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
)

// processField processes a single struct field and returns its schema properties and required status.
// It handles:
// - Type resolution (using Phase 1.1 functions)
// - Tag parsing (using Phase 1.2 functions)
// - Custom models (fields.StructField[T])
// - Validation constraints
func processField(file *ast.File, field *ast.Field) (map[string]spec.Schema, []string, error) {
	if field == nil {
		return nil, nil, nil
	}

	// Skip fields with no name (embedded fields handled separately)
	if len(field.Names) == 0 {
		return nil, nil, nil
	}

	// Skip non-exported fields
	fieldName := field.Names[0].Name
	if !ast.IsExported(fieldName) {
		return nil, nil, nil
	}

	// Parse struct tags
	tags := parseFieldTags(field)

	// Check if field should be ignored
	if tags.Ignore || isSwaggerIgnore(tags.rawTag) {
		return nil, nil, nil
	}

	// Get JSON field name (default to camelCase if not specified)
	jsonName := tags.JSONName
	if jsonName == "" {
		jsonName = toCamelCase(fieldName)
	}

	// Resolve field type
	fieldType := resolveFieldType(field.Type)

	// Check if custom model and extract inner type
	if isCustomModel(fieldType) {
		innerType, err := extractInnerType(fieldType)
		if err == nil && innerType != "" {
			fieldType = innerType

			// If the extracted type doesn't have a package qualifier (no ".")
			// and is not a primitive type, add the package name
			// But DON'T add package to slice/map prefixes - only to the base type
			if file != nil && !strings.Contains(fieldType, ".") && !isPrimitiveTypeName(fieldType) {
				// Handle slices: []Type -> []package.Type
				if strings.HasPrefix(fieldType, "[]") {
					elemType := strings.TrimPrefix(fieldType, "[]")
					if !isPrimitiveTypeName(elemType) && !strings.Contains(elemType, ".") {
						fieldType = "[]" + file.Name.Name + "." + elemType
					}
				} else if strings.HasPrefix(fieldType, "map[") {
					// Handle maps: map[key]value -> map[key]package.value (if needed)
					// For now, leave maps as-is - more complex to handle
				} else {
					// Simple type reference - add package qualifier
					fieldType = file.Name.Name + "." + fieldType
				}
			}
		} else {
			// Extraction failed - this is a fields.StructField[...] we couldn't parse
			// Treat as generic object instead of trying to create invalid reference
			fieldType = "object"
		}
	}

	// If it's still a fields.* named type, resolve it
	if strings.HasPrefix(fieldType, "fields.") {
		if resolvedType := resolveFieldsType(fieldType); resolvedType != "" {
			fieldType = resolvedType
		} else {
			// Could not resolve fields.* type (likely fields.StructField[...] that wasn't extracted)
			// Fall back to generic object
			fieldType = "object"
		}
	}

	// Build property schema
	propSchema := buildPropertySchema(fieldType, tags)

	// Build properties map
	properties := map[string]spec.Schema{
		jsonName: propSchema,
	}

	// Determine if required
	var required []string
	if tags.Required && !tags.OmitEmpty {
		required = append(required, jsonName)
	}

	return properties, required, nil
}

// fieldTags contains parsed tag information
type fieldTags struct {
	JSONName  string
	OmitEmpty bool
	Ignore    bool
	Required  bool
	Min       string
	Max       string
	rawTag    reflect.StructTag
}

// parseFieldTags parses all struct tags from a field
func parseFieldTags(field *ast.Field) fieldTags {
	var tags fieldTags

	if field.Tag == nil {
		return tags
	}

	// Remove backticks from tag value
	tagValue := field.Tag.Value
	if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
		tagValue = tagValue[1 : len(tagValue)-1]
	}

	tags.rawTag = reflect.StructTag(tagValue)

	// Use Phase 1.2 tag parser
	tagInfo := parseCombinedTags(tags.rawTag)

	tags.JSONName = tagInfo.JSONName
	tags.OmitEmpty = tagInfo.OmitEmpty
	tags.Ignore = tagInfo.Ignore
	tags.Required = tagInfo.Required
	tags.Min = tagInfo.Min
	tags.Max = tagInfo.Max

	return tags
}

// resolveFieldType converts AST type expression to type string
func resolveFieldType(expr ast.Expr) string {
	if expr == nil {
		return "object"
	}

	switch t := expr.(type) {
	case *ast.Ident:
		// Basic type like string, int
		return resolveBasicType(t.Name)

	case *ast.SelectorExpr:
		// Package-qualified type like time.Time, uuid.UUID
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgName := ident.Name
			typeName := t.Sel.Name
			fullType := pkgName + "." + typeName
			return resolvePackageType(fullType)
		}
		return "object"

	case *ast.StarExpr:
		// Pointer type - recurse to underlying type
		return resolveFieldType(t.X)

	case *ast.ArrayType:
		// Slice or array type
		return "array"

	case *ast.MapType:
		// Map type
		return "map"

	case *ast.InterfaceType:
		// interface{} or any - no type constraint
		return "interface"

	case *ast.IndexExpr, *ast.IndexListExpr:
		// Generic type like fields.StructField[string]
		// Convert to string representation
		return exprToString(expr)

	default:
		return "object"
	}
}

// resolveBasicType maps Go basic types to OpenAPI types
func resolveBasicType(typeName string) string {
	switch typeName {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "error", "any":
		return "interface"
	default:
		return "object"
	}
}

// resolvePackageType maps package-qualified types to OpenAPI types
func resolvePackageType(fullType string) string {
	// Check for fields.* named types first
	if fieldsType := resolveFieldsType(fullType); fieldsType != "" {
		return fieldsType
	}

	// Special handling for known types
	if fullType == "time.Time" {
		return "string"
	}
	if fullType == "uuid.UUID" {
		return "string"
	}
	if fullType == "decimal.Decimal" {
		return "number"
	}

	// Check if it's a custom model type
	if isCustomModel(fullType) {
		return fullType
	}

	return "object"
}

// exprToString converts an AST expression to string representation
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.IndexExpr:
		// Generic type like Generic[T]
		return exprToString(t.X) + "[" + exprToString(t.Index) + "]"
	default:
		return ""
	}
}

// buildPropertySchema creates an OpenAPI property schema from type and tags
func buildPropertySchema(fieldType string, tags fieldTags) spec.Schema {
	var schema spec.Schema

	// Handle interface types (no type constraint)
	if fieldType == "interface" {
		return schema // Empty schema allows any JSON value
	}

	// Handle array types
	if fieldType == "array" {
		schema.Type = []string{"array"}
		schema.Items = &spec.SchemaOrArray{
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
				},
			},
		}
		return schema
	}

	// Handle map types (object with additionalProperties)
	if fieldType == "map" {
		schema.Type = []string{"object"}
		schema.AdditionalProperties = &spec.SchemaOrBool{
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"string"},
				},
			},
		}
		return schema
	}

	// Handle slice types
	if isSliceType(fieldType) {
		schema.Type = []string{"array"}
		elemType, _ := getSliceElementType(fieldType)
		elemType = stripPointer(elemType)

		// Determine element schema
		var elemSchema *spec.Schema

		// Check if element type is a package-qualified struct type (contains ".")
		// If so, create a reference instead of a generic object
		if strings.Contains(elemType, ".") && !isPrimitiveTypeName(elemType) {
			// This is a qualified type like "classification.JoinedClassification"
			// Create a reference to it
			elemSchema = spec.RefSchema("#/definitions/" + elemType)
		} else {
			// Primitive or local type
			elemSchema = &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{resolveBasicType(elemType)},
				},
			}
		}

		schema.Items = &spec.SchemaOrArray{
			Schema: elemSchema,
		}
		return schema
	}

	// Standard types
	openAPIType := fieldType
	if fieldType == "integer" || fieldType == "number" || fieldType == "string" || fieldType == "boolean" {
		schema.Type = []string{fieldType}
	} else {
		// Check if this is a struct type (contains package qualifier like "account.Properties")
		// If so, create a reference instead of a generic object
		if strings.Contains(fieldType, ".") {
			// This is a package-qualified type - create a $ref
			return *spec.RefSchema("#/definitions/" + fieldType)
		}
		// Other complex types - reference or object
		schema.Type = []string{resolveBasicType(fieldType)}
	}

	// Apply validation constraints
	if tags.Min != "" {
		if openAPIType == "string" {
			if minLen, err := strconv.ParseInt(tags.Min, 10, 64); err == nil {
				schema.MinLength = &minLen
			}
		} else if openAPIType == "integer" || openAPIType == "number" {
			if minVal, err := strconv.ParseFloat(tags.Min, 64); err == nil && !math.IsInf(minVal, 0) && !math.IsNaN(minVal) {
				schema.Minimum = &minVal
			}
		}
	}

	if tags.Max != "" {
		if openAPIType == "string" {
			if maxLen, err := strconv.ParseInt(tags.Max, 10, 64); err == nil {
				schema.MaxLength = &maxLen
			}
		} else if openAPIType == "integer" || openAPIType == "number" {
			if maxVal, err := strconv.ParseFloat(tags.Max, 64); err == nil && !math.IsInf(maxVal, 0) && !math.IsNaN(maxVal) {
				schema.Maximum = &maxVal
			}
		}
	}

	return schema
}

// isPrimitiveTypeName checks if a type name is a Go primitive type
func isPrimitiveTypeName(typeName string) bool {
	primitives := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true, "rune": true,
		"any": true, "interface{}": true,
	}
	return primitives[typeName]
}

// toCamelCase converts PascalCase to camelCase (lowercase first letter)
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	// Convert first character to lowercase
	runes := []rune(s)
	if len(runes) > 0 {
		runes[0] = toLower(runes[0])
	}
	return string(runes)
}

// toLower converts a single rune to lowercase
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
