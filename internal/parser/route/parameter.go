package route

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/griffnb/core-swag/internal/parser/route/domain"
)

var paramPattern = regexp.MustCompile(`(\S+)\s+(\w+)\s+([\S.]+)\s+(\w+)\s+"([^"]+)"`)

// parseParam parses the @param annotation
// Format: @Param name paramType dataType required "description" [Attribute(value)]...
// Example: @Param id path int true "User ID" Format(int64) Minimum(0)
func (s *Service) parseParam(op *operation, line string) error {
	matches := paramPattern.FindStringSubmatch(line)
	if len(matches) != 6 {
		return fmt.Errorf("invalid param format: %s", line)
	}

	name := matches[1]
	paramType := matches[2] // path, query, header, body, formData
	dataType := matches[3]
	requiredStr := strings.ToLower(matches[4])
	description := matches[5]

	required := requiredStr == "true" || requiredStr == "required"

	// Determine if it's an array
	isArray := strings.HasPrefix(dataType, "[]")
	if isArray {
		dataType = strings.TrimPrefix(dataType, "[]")
	}

	// Convert Go types to OpenAPI types
	schemaType, format := convertType(dataType)

	param := domain.Parameter{
		Name:        name,
		In:          paramType,
		Required:    required,
		Description: description,
		Format:      format,
	}

	// Parse attributes after description (Format, Enums, Minimum, Maximum, etc.)
	// Find the end of the description (closing quote) and parse remainder
	matchEnd := len(matches[0])
	if matchEnd < len(line) {
		remainder := line[matchEnd:]
		if err := parseParamAttributes(&param, remainder); err != nil {
			return err
		}
	}

	// Override format if Format() attribute was specified
	if param.Format == "" && format != "" {
		param.Format = format
	}

	if isArray {
		param.Type = "array"
		param.Items = &domain.Items{
			Type:   schemaType,
			Format: param.Format,
		}
	} else {
		param.Type = schemaType
	}

	op.parameters = append(op.parameters, param)
	return nil
}

// parseParam Attributes parses attribute modifiers like Format(int64), Enums(1,2,3), etc.
func parseParamAttributes(param *domain.Parameter, attrs string) error {
	// Regex to match attribute patterns: AttributeName(value)
	attrPattern := regexp.MustCompile(`(\w+)\(([^)]+)\)`)
	matches := attrPattern.FindAllStringSubmatch(attrs, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		attrName := strings.ToLower(match[1])
		attrValue := match[2]

		switch attrName {
		case "format":
			param.Format = attrValue
		case "enums", "enum":
			// Parse comma-separated enum values
			enumStrs := strings.Split(attrValue, ",")
			var enums []interface{}
			for _, e := range enumStrs {
				e = strings.TrimSpace(e)
				// Try to parse as integer first (before checking param.Type)
				var numInt int
				if _, err := fmt.Sscanf(e, "%d", &numInt); err == nil {
					enums = append(enums, float64(numInt)) // JSON uses float64 for numbers
					continue
				}
				// Try as float
				var numFloat float64
				if _, err := fmt.Sscanf(e, "%f", &numFloat); err == nil {
					enums = append(enums, numFloat)
					continue
				}
				// Otherwise treat as string (remove quotes if present)
				e = strings.Trim(e, "\"'")
				enums = append(enums, e)
			}
			param.Enum = enums
		case "minimum", "min":
			var min float64
			if _, err := fmt.Sscanf(attrValue, "%f", &min); err == nil {
				param.Minimum = &min
			}
		case "maximum", "max":
			var max float64
			if _, err := fmt.Sscanf(attrValue, "%f", &max); err == nil {
				param.Maximum = &max
			}
		case "minlength":
			var minLen int
			if _, err := fmt.Sscanf(attrValue, "%d", &minLen); err == nil {
				minLenFloat := float64(minLen)
				param.MinLength = &minLenFloat
			}
		case "maxlength":
			var maxLen int
			if _, err := fmt.Sscanf(attrValue, "%d", &maxLen); err == nil {
				maxLenFloat := float64(maxLen)
				param.MaxLength = &maxLenFloat
			}
		case "default":
			// Parse default value - try numeric first, then boolean, then string
			// Try as integer
			var numInt int
			if _, err := fmt.Sscanf(attrValue, "%d", &numInt); err == nil {
				param.Default = float64(numInt) // JSON uses float64
			} else if numFloat, err := parseFloat(attrValue); err == nil {
				// Try as float
				param.Default = numFloat
			} else if attrValue == "true" {
				// Boolean true
				param.Default = true
			} else if attrValue == "false" {
				// Boolean false
				param.Default = false
			} else {
				// Default to string (remove surrounding quotes if present)
				param.Default = strings.Trim(attrValue, "\"'")
			}
		}
	}

	return nil
}

// parseFloat parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// convertType converts Go types to OpenAPI types
func convertType(goType string) (schemaType string, format string) {
	switch goType {
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer", goType
	case "float32", "float64":
		return "number", goType
	case "bool":
		return "boolean", ""
	case "string":
		return "string", ""
	case "byte":
		return "string", "byte"
	case "rune":
		return "integer", "int32"
	case "object":
		return "object", ""
	case "array":
		return "array", ""
	case "file":
		return "file", ""
	default:
		// For custom types, treat as object
		return "object", ""
	}
}
