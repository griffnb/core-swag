package structparser

import (
	"go/ast"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/console"
)

// processField processes a single struct field and returns its schema properties and required status.
// It handles:
// - Type resolution (using Phase 1.1 functions)
// - Tag parsing (using Phase 1.2 functions)
// - Custom models (fields.StructField[T])
// - Validation constraints
// When public is true, nested struct $ref names get a "Public" suffix appended.
func (s *Service) processField(file *ast.File, field *ast.Field, public bool) (map[string]spec.Schema, []string, error) {
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

	// Skip fields that have NEITHER json NOR column tags
	// This filters out BaseModel private fields like ChangeLogs, Client, etc.
	jsonTag := string(tags.rawTag.Get("json"))
	columnTag := string(tags.rawTag.Get("column"))
	if jsonTag == "" && columnTag == "" {
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
			if file != nil && !strings.Contains(fieldType, ".") && !isPrimitiveTypeName(fieldType) &&
				fieldType != "map" && fieldType != "interface" && fieldType != "object" {
				// Handle slices: []Type -> []package.Type
				if strings.HasPrefix(fieldType, "[]") {
					elemType := strings.TrimPrefix(fieldType, "[]")
					if !isPrimitiveTypeName(elemType) && !strings.Contains(elemType, ".") &&
						elemType != "map" && elemType != "interface" && elemType != "object" &&
						!strings.HasPrefix(elemType, "map[") {
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

	// Check for ConstantField types with enum type parameters before resolving
	// fields.IntConstantField[constants.Role] -> extract "constants.Role" as enum type
	// fields.StringConstantField[constants.GlobalConfigKey] -> extract "constants.GlobalConfigKey" as enum type
	if strings.HasPrefix(fieldType, "fields.") && strings.Contains(fieldType, "ConstantField[") {
		innerType, err := extractInnerType(fieldType)
		if err == nil && innerType != "" {
			// Determine the base OpenAPI type from the field type name
			baseType := "integer"
			if strings.Contains(fieldType, "StringConstantField") {
				baseType = "string"
			}

			// Try to resolve enum values for the inner type
			enumType := innerType
			if file != nil && !strings.Contains(enumType, ".") {
				enumType = file.Name.Name + "." + enumType
			}

			if s.enumLookup != nil && file != nil {
				fullTypeName := s.resolveFullTypeName(enumType, file)
				enumValues, enumErr := s.enumLookup.GetEnumsForType(fullTypeName, file)
				if enumErr == nil && len(enumValues) > 0 {
					// Use registry to get correct definition name (handles NotUnique types)
					refName := s.resolveDefinitionName(enumType, file)
					schema := *spec.RefSchema("#/definitions/" + refName)
					properties := map[string]spec.Schema{jsonName: schema}
					var required []string
					if tags.Required && !tags.OmitEmpty {
						required = append(required, jsonName)
					}
					return properties, required, nil
				}
			}

			// Enum lookup failed or unavailable - fall back to base type
			fieldType = baseType
		} else {
			// Extraction failed - fall back to resolving as primitive
			if resolvedType := resolveFieldsType(fieldType); resolvedType != "" {
				fieldType = resolvedType
			} else {
				fieldType = "object"
			}
		}
	} else if strings.HasPrefix(fieldType, "fields.") {
		// Non-ConstantField fields.* types - resolve to primitives
		if resolvedType := resolveFieldsType(fieldType); resolvedType != "" {
			fieldType = resolvedType
		} else {
			// Could not resolve fields.* type (likely fields.StructField[...] that wasn't extracted)
			// Fall back to generic object
			fieldType = "object"
		}
	}

	// For same-package struct types (no dot, not primitive), add package qualifier
	if file != nil && !strings.Contains(fieldType, ".") &&
		!isPrimitiveTypeName(fieldType) &&
		!strings.HasPrefix(fieldType, "[]") && !strings.HasPrefix(fieldType, "map[") && // Don't qualify arrays/maps
		fieldType != "object" && fieldType != "array" && fieldType != "map" && fieldType != "interface" &&
		fieldType != "integer" && fieldType != "number" && fieldType != "string" && fieldType != "boolean" {
		// This is a same-package struct type like "Properties"
		// Add package qualifier
		fieldType = file.Name.Name + "." + fieldType
	}

	// Build property schema
	propSchema := s.buildPropertySchema(fieldType, tags, file, public)

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
		// Slice or array type - preserve element type
		elemType := resolveFieldType(t.Elt)
		return "[]" + elemType

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
		// Preserve the type name for struct types (don't convert to "object")
		// This allows processField to properly add package qualifiers
		return typeName
	}
}

// resolvePackageType maps package-qualified types to OpenAPI types
func resolvePackageType(fullType string) string {
	// Check for fields.* named types first
	if fieldsType := resolveFieldsType(fullType); fieldsType != "" {
		return fieldsType
	}

	// Check if it's an extended primitive (time.Time, UUID, decimal)
	// Return the qualified name so buildPropertySchema can properly detect it
	if isExtendedPrimitive(fullType) {
		return fullType // Return full type name (e.g., "types.UUID")
	}

	// For any package-qualified type (constants.ClassificationType, account.Properties, etc.)
	// Return the full type name so buildPropertySchema can create a $ref
	// This handles both enum types and struct types from other packages
	return fullType
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
	case *ast.MapType:
		return "map"
	case *ast.InterfaceType:
		return "interface"
	case *ast.IndexExpr:
		// Generic type like Generic[T]
		return exprToString(t.X) + "[" + exprToString(t.Index) + "]"
	default:
		return ""
	}
}

// buildPropertySchema creates an OpenAPI property schema from type and tags.
// When public is true, nested struct $ref names get a "Public" suffix appended.
func (s *Service) buildPropertySchema(fieldType string, tags fieldTags, file *ast.File, public bool) spec.Schema {
	var schema spec.Schema

	// Handle interface types (no type constraint) - treat as object
	if fieldType == "interface" {
		schema.Type = []string{"object"}
		return schema
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

		// Check if element type is a primitive (basic or extended)
		if isPrimitiveTypeName(elemType) || isExtendedPrimitive(elemType) {
			// It's a primitive - get the proper schema with format
			baseType, format := getPrimitiveSchema(elemType)
			elemSchema = &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{baseType},
				},
			}
			if format != "" {
				elemSchema.Format = format
			}
		} else if strings.Contains(elemType, ".") {
			// Check if element type is a package-qualified struct type (contains ".")
			// For enum types, use registry to get correct definition name (handles NotUnique types)
			// For struct types, keep short name since StructParserService stores definitions that way
			refName := elemType
			if s.isStructType(elemType, file) {
				if public {
					refName = elemType + "Public"
				}
			} else {
				refName = s.resolveDefinitionName(elemType, file)
			}
			elemSchema = spec.RefSchema("#/definitions/" + refName)
		} else if elemType == "map" || elemType == "interface" || elemType == "object" ||
			strings.HasPrefix(elemType, "map[") {
			// Meta-types - treat as generic object
			elemSchema = &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
				},
			}
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
		// Check if this is an extended primitive (time.Time, types.UUID, etc.) BEFORE checking for refs
		// Extended primitives have dots but should NOT be refs
		if isPrimitiveTypeName(fieldType) || isExtendedPrimitive(fieldType) {
			// It's a primitive - get the proper schema
			baseType, format := getPrimitiveSchema(fieldType)
			schema.Type = []string{baseType}
			if format != "" {
				schema.Format = format
			}
		} else if strings.Contains(fieldType, ".") {
			// Check if this is a struct type (contains package qualifier like "account.Properties")
			// First check if it's an enum type
			console.Logger.Debug(">>> buildPropertySchema: checking enum for fieldType: %s, enumLookup nil? %v, file nil? %v\n", fieldType, s.enumLookup == nil, file == nil)
			if s.enumLookup != nil && file != nil {
				// Resolve the full package path for the type
				fullTypeName := s.resolveFullTypeName(fieldType, file)
				console.Logger.Debug(">>> buildPropertySchema: resolved %s to %s\n", fieldType, fullTypeName)

				enumValues, err := s.enumLookup.GetEnumsForType(fullTypeName, file)
				console.Logger.Debug(">>> buildPropertySchema: GetEnumsForType returned %d values, err=%v\n", len(enumValues), err)
				if err == nil && len(enumValues) > 0 {
					// This is an enum type - create a $ref to the enum definition
					// Enums don't get Public suffix since enum values are identical regardless of public context
					// Use registry to get correct definition name (handles NotUnique types)
					refName := s.resolveDefinitionName(fieldType, file)
					return *spec.RefSchema("#/definitions/" + refName)
				}
			}
			// Not an enum - only add Public suffix if the type is a struct.
			// Constants, type aliases to primitives, etc. don't have Public variants.
			// For struct types, keep short name since StructParserService stores definitions that way.
			// For non-struct types, use registry to get correct definition name (handles NotUnique types).
			refName := fieldType
			if s.isStructType(fieldType, file) {
				if public {
					refName = fieldType + "Public"
				}
			} else {
				refName = s.resolveDefinitionName(fieldType, file)
			}
			return *spec.RefSchema("#/definitions/" + refName)
		} else {
			// Other complex types - reference or object
			schema.Type = []string{resolveBasicType(fieldType)}
		}
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

// isExtendedPrimitive checks if a type is an extended primitive (time.Time, UUID, decimal)
// These types have package qualifiers but should be treated as primitives, not refs
func isExtendedPrimitive(typeName string) bool {
	// Strip pointer prefix
	cleanType := strings.TrimPrefix(typeName, "*")

	extendedPrimitives := map[string]bool{
		"time.Time":                              true,
		"decimal.Decimal":                        true,
		"github.com/shopspring/decimal.Decimal":  true,
		"types.UUID":                             true,
		"uuid.UUID":                              true,
		"github.com/griffnb/core/lib/types.UUID": true,
		"github.com/google/uuid.UUID":            true,
	}

	return extendedPrimitives[cleanType]
}

// getPrimitiveSchema returns the OpenAPI type and format for a primitive type
func getPrimitiveSchema(typeName string) (schemaType string, format string) {
	// Strip pointer prefix
	cleanType := strings.TrimPrefix(typeName, "*")

	switch cleanType {
	case "time.Time":
		return "string", "date-time"
	case "types.UUID", "uuid.UUID", "github.com/griffnb/core/lib/types.UUID", "github.com/google/uuid.UUID":
		return "string", "uuid"
	case "decimal.Decimal", "github.com/shopspring/decimal.Decimal":
		return "number", ""
	default:
		return resolveBasicType(typeName), ""
	}
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

// isStructType checks if a package-qualified type name (e.g., "account.Properties")
// resolves to a struct type via the registry. Returns false for enums, type aliases
// to primitives, and any type that cannot be confirmed as a struct.
func (s *Service) isStructType(typeName string, file *ast.File) bool {
	if s.registry == nil || file == nil {
		return false
	}

	typeDef := s.registry.FindTypeSpec(typeName, file)
	if typeDef == nil || typeDef.TypeSpec == nil {
		return false
	}

	_, isStruct := typeDef.TypeSpec.Type.(*ast.StructType)
	return isStruct
}
