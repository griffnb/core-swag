package structparser

import (
	"reflect"
	"strings"
)

// TagInfo contains all parsed information from struct field tags
type TagInfo struct {
	JSONName   string // Field name from json tag
	OmitEmpty  bool   // Whether json tag has omitempty
	Ignore     bool   // Whether field should be ignored (json:"-")
	Visibility string // Visibility level: "view", "edit", or "private"
	Required   bool   // Whether field is required (from binding/validate tags)
	Optional   bool   // Whether field is explicitly optional
	Min        string // Minimum value/length constraint
	Max        string // Maximum value/length constraint
}

// parseJSONTag parses the json struct tag and returns field name, omitempty flag, and ignore flag.
// If no json tag is present, falls back to column tag for custom model systems.
//
// Examples:
//   - `json:"first_name"` → ("first_name", false, false)
//   - `json:"count,omitempty"` → ("count", true, false)
//   - `json:"-"` → ("", false, true)
//   - `column:"external_id"` (no json tag) → ("external_id", false, false)
func parseJSONTag(tag reflect.StructTag) (name string, omitEmpty bool, ignore bool) {
	jsonTag := tag.Get("json")
	if jsonTag == "" {
		// Fallback to column tag for custom model systems
		columnTag := tag.Get("column")
		if columnTag != "" {
			return strings.TrimSpace(columnTag), false, false
		}
		return "", false, false
	}

	// Split by comma to separate name from options
	parts := strings.Split(jsonTag, ",")

	// First part is the field name
	name = strings.TrimSpace(parts[0])

	// Check if field should be ignored
	if name == "-" {
		return "", false, true
	}

	// Check for omitempty in remaining parts
	for i := 1; i < len(parts); i++ {
		if strings.TrimSpace(parts[i]) == "omitempty" {
			omitEmpty = true
			break
		}
	}

	return name, omitEmpty, ignore
}

// parsePublicTag parses the public struct tag and returns the visibility level.
//
// Examples:
//   - `public:"view"` → "view"
//   - `public:"edit"` → "edit"
//   - No tag or invalid value → "private"
func parsePublicTag(tag reflect.StructTag) (visibility string) {
	publicTag := tag.Get("public")
	if publicTag == "" {
		return "private"
	}

	// Normalize to lowercase and trim spaces
	visibility = strings.ToLower(strings.TrimSpace(publicTag))

	// Only accept "view" or "edit" as valid values
	if visibility == "view" || visibility == "edit" {
		return visibility
	}

	// Invalid value defaults to private
	return "private"
}

// parseValidationTags parses binding and validate struct tags and returns validation constraints.
//
// Examples:
//   - `binding:"required"` → (true, false, "", "")
//   - `validate:"required,min=1,max=100"` → (true, false, "1", "100")
//   - `validate:"optional"` → (false, true, "", "")
func parseValidationTags(tag reflect.StructTag) (required bool, optional bool, min string, max string) {
	// Parse both binding and validate tags
	bindingTag := tag.Get("binding")
	validateTag := tag.Get("validate")

	// Combine both tags for parsing
	allValidation := bindingTag
	if validateTag != "" {
		if allValidation != "" {
			allValidation += "," + validateTag
		} else {
			allValidation = validateTag
		}
	}

	if allValidation == "" {
		return false, false, "", ""
	}

	// Split by comma to get individual validation rules
	rules := strings.Split(allValidation, ",")

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		// Check for required/optional
		if rule == "required" {
			required = true
		} else if rule == "optional" {
			optional = true
		} else if strings.HasPrefix(rule, "min=") || strings.HasPrefix(rule, "gte=") {
			// Extract min value
			parts := strings.SplitN(rule, "=", 2)
			if len(parts) == 2 {
				min = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(rule, "max=") || strings.HasPrefix(rule, "lte=") {
			// Extract max value
			parts := strings.SplitN(rule, "=", 2)
			if len(parts) == 2 {
				max = strings.TrimSpace(parts[1])
			}
		}
	}

	return required, optional, min, max
}

// parseCombinedTags parses all struct tags together and returns a TagInfo with all parsed data.
//
// This is the main entry point that combines all individual tag parsers.
//
// Example:
//   - `json:"username" public:"view" validate:"required,min=3,max=20"` →
//     TagInfo{JSONName: "username", Visibility: "view", Required: true, Min: "3", Max: "20"}
func parseCombinedTags(tag reflect.StructTag) TagInfo {
	// Parse each tag type
	jsonName, omitEmpty, ignore := parseJSONTag(tag)
	visibility := parsePublicTag(tag)
	required, optional, min, max := parseValidationTags(tag)

	return TagInfo{
		JSONName:   jsonName,
		OmitEmpty:  omitEmpty,
		Ignore:     ignore,
		Visibility: visibility,
		Required:   required,
		Optional:   optional,
		Min:        min,
		Max:        max,
	}
}

// isSwaggerIgnore checks if the field has swaggerignore:"true" tag.
//
// Examples:
//   - `swaggerignore:"true"` → true
//   - `swaggerignore:"false"` → false
//   - No tag → false
func isSwaggerIgnore(tag reflect.StructTag) bool {
	ignoreTag := tag.Get("swaggerignore")
	if ignoreTag == "" {
		return false
	}

	// Case-insensitive comparison with trimming
	return strings.EqualFold(strings.TrimSpace(ignoreTag), "true")
}

// extractEnumValues extracts enum values from oneof validation tag.
//
// Examples:
//   - `validate:"oneof=red green blue"` → ["red", "green", "blue"]
//   - `validate:"oneof='value 1' 'value 2'"` → ["value 1", "value 2"]
//   - No oneof tag → nil
func extractEnumValues(tag reflect.StructTag) []string {
	validateTag := tag.Get("validate")
	if validateTag == "" {
		return nil
	}

	// Find the oneof rule
	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if strings.HasPrefix(rule, "oneof=") {
			// Extract the value part after "oneof="
			valuesPart := strings.TrimPrefix(rule, "oneof=")
			if valuesPart == "" {
				return nil
			}

			// Parse values, handling quoted strings
			return parseOneOfValues(valuesPart)
		}
	}

	return nil
}

// parseOneOfValues parses space-separated values, handling single-quoted strings.
// Examples:
//   - "red green blue" → ["red", "green", "blue"]
//   - "'value 1' 'value 2'" → ["value 1", "value 2"]
func parseOneOfValues(input string) []string {
	var values []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(input); i++ {
		char := input[i]

		if char == '\'' {
			// Toggle quote mode
			inQuote = !inQuote
		} else if char == ' ' && !inQuote {
			// Space outside quotes - end current value
			if current.Len() > 0 {
				values = append(values, current.String())
				current.Reset()
			}
		} else {
			// Regular character - add to current value
			current.WriteByte(char)
		}
	}

	// Add last value if any
	if current.Len() > 0 {
		values = append(values, current.String())
	}

	return values
}
