package structparser

import (
	"errors"
	"strings"
	"unicode"
)

// normalizeGenericTypeName normalizes type names for use as identifiers.
// Replaces dots with underscores.
// Example: "model.Account" -> "model_Account"
func normalizeGenericTypeName(typeName string) string {
	return strings.Replace(typeName, ".", "_", -1)
}

// stripPointer removes leading asterisks from type names.
// Example: "*model.Account" -> "model.Account"
// Example: "**string" -> "string"
func stripPointer(typeName string) string {
	return strings.TrimLeft(typeName, "*")
}

// isSliceType checks if a type name represents a slice.
// Example: "[]string" -> true
// Example: "[]*model.User" -> true
// Example: "string" -> false
func isSliceType(typeName string) bool {
	return strings.HasPrefix(typeName, "[]")
}

// isMapType checks if a type name represents a map.
// Example: "map[string]int" -> true
// Example: "string" -> false
func isMapType(typeName string) bool {
	return strings.HasPrefix(typeName, "map[")
}

// isCustomModel checks if a type is a custom model wrapper like fields.StructField.
// Returns true for "fields.StructField" and "fields.StructField[T]"
func isCustomModel(typeName string) bool {
	return strings.HasPrefix(typeName, "fields.StructField")
}

// resolveFieldsType maps fields.* named types to OpenAPI types.
// These are concrete field types (not generic) used in the custom model system.
// Examples:
//   - "fields.StringField" -> "string"
//   - "fields.IntField" -> "integer"
//   - "fields.BoolField" -> "boolean"
//   - "fields.UUIDField" -> "string"
//   - "fields.FloatField" -> "number"
//   - "fields.IntConstantField" -> "integer"
//   - "fields.StringConstantField" -> "string"
//
// Returns empty string if not a recognized fields.* type.
func resolveFieldsType(typeName string) string {
	if !strings.HasPrefix(typeName, "fields.") {
		return ""
	}

	// Extract the field type name after "fields."
	fieldType := strings.TrimPrefix(typeName, "fields.")

	// Map to OpenAPI types
	switch {
	case strings.HasPrefix(fieldType, "String"):
		return "string"
	case strings.HasPrefix(fieldType, "Int"):
		return "integer"
	case strings.HasPrefix(fieldType, "Bool"):
		return "boolean"
	case strings.HasPrefix(fieldType, "Float") || strings.HasPrefix(fieldType, "Decimal"):
		return "number"
	case strings.HasPrefix(fieldType, "UUID"):
		return "string"
	case strings.HasPrefix(fieldType, "Time"):
		return "string"
	case strings.HasPrefix(fieldType, "Struct"):
		// fields.StructField[T] is a generic type - caller should extract T
		return ""
	default:
		// Unknown fields.* type - treat as object
		return "object"
	}
}

// getSliceElementType extracts the element type from a slice type.
// Example: "[]string" -> "string"
// Example: "[]*model.User" -> "*model.User"
// Example: "[][]int" -> "[]int"
func getSliceElementType(typeName string) (string, error) {
	if !isSliceType(typeName) {
		return "", errors.New("not a slice type")
	}
	return typeName[2:], nil
}

// splitGenericTypeName splits a generic type name into base type and parameters.
// Example: "fields.StructField[string]" -> ("fields.StructField", ["string"])
// Ported from legacy generics.go
func splitGenericTypeName(fullGenericForm string) (string, []string) {
	// Remove all spaces
	fullGenericForm = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, fullGenericForm)

	// Check if it ends with ']'
	if len(fullGenericForm) == 0 || fullGenericForm[len(fullGenericForm)-1] != ']' {
		return "", nil
	}

	// Split at the first '[' and remove the last ']'
	genericParams := strings.SplitN(fullGenericForm[:len(fullGenericForm)-1], "[", 2)
	if len(genericParams) == 1 {
		return "", nil
	}

	// Generic type name
	genericTypeName := genericParams[0]

	// Parse parameters respecting bracket depth
	depth := 0
	params := strings.FieldsFunc(genericParams[1], func(r rune) bool {
		if r == '[' {
			depth++
		} else if r == ']' {
			depth--
		} else if r == ',' && depth == 0 {
			return true
		}
		return false
	})

	if depth != 0 {
		return "", nil
	}

	return genericTypeName, params
}

// extractInnerType extracts the inner type from a generic wrapper.
// Example: "fields.StructField[string]" -> "string"
// Example: "fields.StructField[*model.Account]" -> "model.Account" (strips pointer)
// Example: "fields.StructField[[]*model.User]" -> "[]model.User" (strips pointer from slice elements)
func extractInnerType(typeName string) (string, error) {
	// Try to split as generic type
	baseName, params := splitGenericTypeName(typeName)

	// If it's a generic type with parameters, extract the first parameter
	if baseName != "" && len(params) > 0 {
		innerType := params[0]

		// Handle slice of pointers: []*Type -> []Type
		if isSliceType(innerType) {
			elemType, err := getSliceElementType(innerType)
			if err == nil {
				elemType = stripPointer(elemType)
				innerType = "[]" + elemType
			}
		} else {
			// Strip pointer if present (for non-slice types)
			innerType = stripPointer(innerType)
		}

		return innerType, nil
	}

	// Not a generic type, return as-is
	return typeName, nil
}
