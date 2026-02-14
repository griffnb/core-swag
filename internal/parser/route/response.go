package route

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/griffnb/core-swag/internal/parser/route/domain"
)

var (
	// Matches: 200 {object} string "description" OR 200 {object} string
	responsePattern = regexp.MustCompile(`([\w,]+)\s+\{(\w+)\}\s+(\S+)(?:\s+"([^"]+)")?`)
	// Matches: 200 "description"
	emptyResponsePattern = regexp.MustCompile(`([\w,]+)\s+"([^"]+)"`)
)

// parseResponse parses @success, @failure, or @response annotations
func (s *Service) parseResponse(op *operation, line string) error {
	// Try to match with schema first
	matches := responsePattern.FindStringSubmatch(line)
	if len(matches) == 5 {
		return s.parseResponseWithSchema(op, matches)
	}

	// Try to match without schema (just status code and description)
	matches = emptyResponsePattern.FindStringSubmatch(line)
	if len(matches) == 3 {
		return s.parseEmptyResponse(op, matches)
	}

	return fmt.Errorf("invalid response format: %s", line)
}

// parseResponseWithSchema parses a response with a schema
func (s *Service) parseResponseWithSchema(op *operation, matches []string) error {
	statusCodes := matches[1]
	schemaType := matches[2] // object or array
	dataType := matches[3]
	description := ""
	if len(matches) > 4 && matches[4] != "" {
		description = matches[4]
	} else {
		description = "OK" // Default description
	}

	// Build the schema with package context for type qualification
	schema := s.buildSchemaWithPackage(schemaType, dataType, op.packageName)

	// Parse status codes (can be comma-separated)
	for _, codeStr := range strings.Split(statusCodes, ",") {
		codeStr = strings.TrimSpace(codeStr)

		code, err := strconv.Atoi(codeStr)
		if err != nil {
			return fmt.Errorf("invalid status code: %s", codeStr)
		}

		// Create or update the response
		response := domain.Response{
			Description: description,
			Schema:      schema,
			Headers:     make(map[string]domain.Header),
		}

		// Preserve existing headers if response already exists
		if existing, ok := op.responses[code]; ok {
			response.Headers = existing.Headers
		}

		op.responses[code] = response
	}

	return nil
}

// parseEmptyResponse parses a response without a schema
func (s *Service) parseEmptyResponse(op *operation, matches []string) error {
	statusCodes := matches[1]
	description := matches[2]

	// Parse status codes (can be comma-separated)
	for _, codeStr := range strings.Split(statusCodes, ",") {
		codeStr = strings.TrimSpace(codeStr)

		code, err := strconv.Atoi(codeStr)
		if err != nil {
			return fmt.Errorf("invalid status code: %s", codeStr)
		}

		// Create or update the response
		response := domain.Response{
			Description: description,
			Headers:     make(map[string]domain.Header),
		}

		// Preserve existing headers if response already exists
		if existing, ok := op.responses[code]; ok {
			response.Headers = existing.Headers
		}

		op.responses[code] = response
	}

	return nil
}

// buildSchema builds a schema from the schemaType and dataType
func (s *Service) buildSchema(schemaType, dataType string) *domain.Schema {
	return s.buildSchemaWithPackage(schemaType, dataType, "")
}

// buildSchemaWithPackage builds a schema with package qualification for custom types
func (s *Service) buildSchemaWithPackage(schemaType, dataType, packageName string) *domain.Schema {
	schema := &domain.Schema{}

	if schemaType == "array" {
		schema.Type = "array"
		// For array items, check if it's a model type
		itemSchema := s.buildSchemaForTypeWithPackage(dataType, packageName)
		schema.Items = itemSchema
	} else {
		// Build schema for the type
		return s.buildSchemaForTypeWithPackage(dataType, packageName)
	}

	return schema
}

// buildSchemaForType builds a schema for a single type, creating refs for model types
func (s *Service) buildSchemaForType(dataType string) *domain.Schema {
	return s.buildSchemaForTypeWithPackage(dataType, "")
}

// buildSchemaForTypeWithPackage builds a schema for a single type with package qualification
func (s *Service) buildSchemaForTypeWithPackage(dataType, packageName string) *domain.Schema {
	// Check if it's a primitive type
	primitiveType := convertTypeToSchemaType(dataType)
	if primitiveType != "object" {
		// It's a primitive - return with type
		return &domain.Schema{Type: primitiveType}
	}

	// It's a custom type - create a reference
	// If dataType already contains a dot (package.Type), use it as-is
	// Otherwise, qualify it with the packageName
	qualifiedType := dataType
	if packageName != "" && !strings.Contains(dataType, ".") {
		qualifiedType = packageName + "." + dataType
	}

	ref := "#/definitions/" + qualifiedType
	return &domain.Schema{Ref: ref}
}

// convertTypeToSchemaType converts a data type to a schema type
func convertTypeToSchemaType(dataType string) string {
	switch dataType {
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "string":
		return "string"
	default:
		// For custom types, treat as object
		return "object"
	}
}

// parseHeader parses @header annotation
// Format: @Header statusCode {type} headerName "description"
// Example: @Header 200 {string} X-Request-Id "Request ID"
func (s *Service) parseHeader(op *operation, line string) error {
	// Reuse response pattern to parse header
	matches := responsePattern.FindStringSubmatch(line)
	if len(matches) != 5 {
		return fmt.Errorf("invalid header format: %s", line)
	}

	statusCodes := matches[1]
	headerType := matches[2]
	headerName := matches[3]
	description := matches[4]

	header := domain.Header{
		Type:        convertTypeToSchemaType(headerType),
		Description: description,
	}

	// Handle "all" status code
	if strings.EqualFold(statusCodes, "all") {
		// Add header to all existing responses
		for code, response := range op.responses {
			if response.Headers == nil {
				response.Headers = make(map[string]domain.Header)
			}
			response.Headers[headerName] = header
			op.responses[code] = response
		}
		return nil
	}

	// Parse specific status codes
	for _, codeStr := range strings.Split(statusCodes, ",") {
		codeStr = strings.TrimSpace(codeStr)

		code, err := strconv.Atoi(codeStr)
		if err != nil {
			return fmt.Errorf("invalid status code: %s", codeStr)
		}

		// Get or create the response
		response, ok := op.responses[code]
		if !ok {
			response = domain.Response{
				Headers: make(map[string]domain.Header),
			}
		}

		if response.Headers == nil {
			response.Headers = make(map[string]domain.Header)
		}

		response.Headers[headerName] = header
		op.responses[code] = response
	}

	return nil
}
