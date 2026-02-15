package schema

import (
	"fmt"
	"strings"

	"github.com/go-openapi/spec"
)

// parseFieldOverrides parses field override syntax: "field1=Type1,field2=Type2"
// Returns map of field names to type strings.
// Handles nested braces and respects bracket depth.
//
// Examples:
//   - "data=Account" → {"data": "Account"}
//   - "data=Account,meta=Meta" → {"data": "Account", "meta": "Meta"}
//   - "data=Inner{field=Type}" → {"data": "Inner{field=Type}"}
func parseFieldOverrides(s string) (map[string]string, error) {
	if s == "" {
		return map[string]string{}, nil
	}

	// Split on commas, respecting bracket depth
	fields := splitFields(s)
	result := make(map[string]string)

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		// Split on equals sign
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field override: %s", field)
		}

		fieldName := strings.TrimSpace(parts[0])
		typeName := strings.TrimSpace(parts[1])

		if fieldName == "" {
			return nil, fmt.Errorf("empty field name in: %s", field)
		}
		if typeName == "" {
			return nil, fmt.Errorf("empty type name in: %s", field)
		}

		result[fieldName] = typeName
	}

	return result, nil
}

// splitFields splits a string on commas, respecting bracket depth.
// This ensures "data=Inner{field=Type},meta=Meta" splits correctly.
func splitFields(s string) []string {
	var fields []string
	var current strings.Builder
	nestLevel := 0

	for _, char := range s {
		switch char {
		case '{':
			nestLevel++
			current.WriteRune(char)
		case '}':
			nestLevel--
			current.WriteRune(char)
		case ',':
			if nestLevel == 0 {
				// Top-level comma, split here
				if current.Len() > 0 {
					fields = append(fields, current.String())
					current.Reset()
				}
			} else {
				// Inside braces, keep the comma
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Add remaining content
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

// parseCombinedType extracts base type and field overrides from combined syntax.
// Input: "BaseType{field1=Type1,field2=Type2}"
// Returns: baseType, overrides map, error
//
// Examples:
//   - "Response{data=Account}" → "Response", {"data": "Account"}, nil
//   - "response.SuccessResponse{data=account.Account}" → "response.SuccessResponse", {"data": "account.Account"}, nil
//   - "Response" → "Response", {}, nil (no overrides)
//   - "Response{}" → "Response", {}, nil (empty overrides)
func parseCombinedType(refType string) (string, map[string]string, error) {
	if refType == "" {
		return "", nil, fmt.Errorf("empty type")
	}

	// Find opening brace
	openIdx := strings.Index(refType, "{")
	if openIdx == -1 {
		// No braces, just return the type name
		return refType, map[string]string{}, nil
	}

	// Extract base type (before opening brace)
	baseType := refType[:openIdx]

	// Validate format - must end with closing brace
	if !strings.HasSuffix(refType, "}") {
		return "", nil, fmt.Errorf("missing closing brace in: %s", refType)
	}

	// Count braces to detect extra closing braces
	openCount := strings.Count(refType[openIdx:], "{")
	closeCount := strings.Count(refType[openIdx:], "}")
	if closeCount > openCount {
		return "", nil, fmt.Errorf("invalid format: extra closing brace in: %s", refType)
	}

	// Extract override section (between first { and last })
	closeIdx := strings.LastIndex(refType, "}")
	overrideSection := refType[openIdx+1 : closeIdx]

	// Parse field overrides
	overrides, err := parseFieldOverrides(overrideSection)
	if err != nil {
		return "", nil, err
	}

	return baseType, overrides, nil
}

// shouldUseAllOf determines if AllOf composition is needed.
// Returns false if no overrides or if base can be merged directly.
//
// Logic:
//   - No overrides → false (return base unchanged)
//   - Empty object base (no ref, no properties) → false (merge properties directly)
//   - Ref with overrides → true (use AllOf)
//   - Object with properties and overrides → true (use AllOf)
func shouldUseAllOf(baseSchema *spec.Schema, overrideProperties map[string]spec.Schema) bool {
	// No base or no overrides → don't use AllOf
	if baseSchema == nil || len(overrideProperties) == 0 {
		return false
	}

	// Check if base is empty object (no ref, no properties, type = "object")
	// In this case, merge properties directly without AllOf
	hasRef := baseSchema.Ref.GetURL() != nil
	hasProperties := len(baseSchema.Properties) > 0
	isObjectType := len(baseSchema.Type) > 0 && baseSchema.Type[0] == "object"

	if !hasRef && !hasProperties && isObjectType {
		// Empty object base - merge properties directly (no AllOf)
		return false
	}

	// Base has ref or properties, and we have overrides → use AllOf
	return true
}

// buildAllOfSchema creates an AllOf schema combining base schema with property overrides.
// If base is empty object, merges properties directly without AllOf.
// If no overrides, returns base unchanged.
//
// Examples:
//   - Ref + overrides → AllOf with two schemas
//   - Empty object + overrides → Base with properties merged (no AllOf)
//   - Any base + no overrides → Base unchanged (no AllOf)
func buildAllOfSchema(baseSchema *spec.Schema, overrideProperties map[string]spec.Schema) *spec.Schema {
	// Handle nil base
	if baseSchema == nil {
		return &spec.Schema{}
	}

	// No overrides - return base unchanged
	if len(overrideProperties) == 0 {
		return baseSchema
	}

	// Check if we should use AllOf composition
	if !shouldUseAllOf(baseSchema, overrideProperties) {
		// Empty object - merge properties directly
		baseSchema.Properties = overrideProperties
		return baseSchema
	}

	// Create AllOf composition
	// First element: base schema
	// Second element: object with property overrides
	return spec.ComposedSchema(*baseSchema, spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: overrideProperties,
		},
	})
}
