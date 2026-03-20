package route

import (
	"fmt"
	"go/ast"
	"regexp"
	"strconv"
	"strings"

	routedomain "github.com/griffnb/core-swag/internal/parser/route/domain"
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
	var description string
	if len(matches) > 4 && matches[4] != "" {
		description = matches[4]
	} else {
		description = "OK" // Default description
	}

	// Build the schema with package context and @Public support
	schema := s.buildSchemaWithPackageAndPublic(schemaType, dataType, op.packageName, op.isPublic, op.astFile)

	// Parse status codes (can be comma-separated)
	for _, codeStr := range strings.Split(statusCodes, ",") {
		codeStr = strings.TrimSpace(codeStr)

		code, err := strconv.Atoi(codeStr)
		if err != nil {
			return fmt.Errorf("invalid status code: %s", codeStr)
		}

		// Create or update the response
		response := routedomain.Response{
			Description: description,
			Schema:      schema,
			Headers:     make(map[string]routedomain.Header),
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
		response := routedomain.Response{
			Description: description,
			Headers:     make(map[string]routedomain.Header),
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
func (s *Service) buildSchema(schemaType, dataType string) *routedomain.Schema {
	return s.buildSchemaWithPackage(schemaType, dataType, "")
}

// buildSchemaWithPackage builds a schema with package qualification for custom types
func (s *Service) buildSchemaWithPackage(schemaType, dataType, packageName string) *routedomain.Schema {
	return s.buildSchemaWithPackageAndPublic(schemaType, dataType, packageName, false, nil)
}

// buildSchemaWithPackageAndPublic builds a schema with package qualification and @Public support.
// file is used for import resolution to produce fully qualified TypePath values.
func (s *Service) buildSchemaWithPackageAndPublic(schemaType, dataType, packageName string, isPublic bool, file *ast.File) *routedomain.Schema {
	schema := &routedomain.Schema{}

	// Check for AllOf combined type syntax: Response{data=Account}
	if strings.Contains(dataType, "{") {
		// Use AllOf composition
		return s.buildAllOfResponseSchema(dataType, packageName, isPublic, file)
	}

	if schemaType == "file" {
		// File response (e.g., @Success 200 {file} []byte "File content")
		schema.Type = "file"
		return schema
	} else if schemaType == "array" {
		schema.Type = "array"
		// For array items, apply @Public flag
		itemSchema := s.buildSchemaForTypeWithPublic(dataType, packageName, isPublic, file)
		schema.Items = itemSchema
	} else {
		// Build schema for the type with @Public flag
		return s.buildSchemaForTypeWithPublic(dataType, packageName, isPublic, file)
	}

	return schema
}

// buildSchemaForType builds a schema for a single type, creating refs for model types
func (s *Service) buildSchemaForType(dataType string) *routedomain.Schema {
	return s.buildSchemaForTypeWithPackage(dataType, "")
}

// buildSchemaForTypeWithPackage builds a schema for a single type with package qualification
func (s *Service) buildSchemaForTypeWithPackage(dataType, packageName string) *routedomain.Schema {
	return s.buildSchemaForTypeWithPublic(dataType, packageName, false, nil)
}

// buildSchemaForTypeWithPublic builds a schema for a single type with optional Public suffix.
// file is used for import resolution to produce fully qualified TypePath values.
func (s *Service) buildSchemaForTypeWithPublic(dataType, packageName string, isPublic bool, file *ast.File) *routedomain.Schema {
	// Check if it's a primitive type
	primitiveType := convertTypeToSchemaType(dataType)
	if primitiveType != "object" {
		// It's a primitive - return with type
		return &routedomain.Schema{Type: primitiveType}
	}

	// Wildcard types default to object, not a $ref
	if dataType == "any" || dataType == "interface{}" || dataType == "object" {
		return &routedomain.Schema{Type: "object"}
	}

	// It's a custom type - create a reference
	// If dataType already contains a dot (package.Type), use it as-is
	// Otherwise, qualify it with the packageName
	qualifiedType := dataType
	if packageName != "" && !strings.Contains(dataType, ".") {
		qualifiedType = packageName + "." + dataType
	}

	// Resolve full import path for unambiguous registry lookup.
	// Do this before appending Public suffix since the registry stores base types.
	typePath := s.resolveTypePath(qualifiedType, file)

	// If @Public annotation is present, only append Public suffix for struct types.
	// Enums and other non-struct types are identical regardless of public context.
	if isPublic && !s.hasNoPublicAnnotation(qualifiedType) && s.isStructType(qualifiedType) {
		qualifiedType = qualifiedType + "Public"
		if typePath != "" {
			typePath += "Public"
		}
	}

	ref := "#/definitions/" + qualifiedType
	return &routedomain.Schema{Ref: ref, TypePath: typePath}
}

// convertTypeToSchemaType converts a data type to a schema type.
// Handles basic Go types and extended primitives (time.Time, UUID, decimal).
func convertTypeToSchemaType(dataType string) string {
	// Strip pointer prefix for processing
	cleanType := strings.TrimPrefix(dataType, "*")

	switch cleanType {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "string", "[]byte", "[]uint8":
		return "string"
	// Extended primitives
	case "time.Time":
		return "string" // with format: date-time
	case "types.UUID", "uuid.UUID", "github.com/griffnb/core/lib/types.UUID", "github.com/google/uuid.UUID":
		return "string" // with format: uuid
	case "types.URN", "github.com/griffnb/core/lib/types.URN":
		return "string" // with format: uri
	case "decimal.Decimal", "github.com/shopspring/decimal.Decimal":
		return "number"
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

	header := routedomain.Header{
		Type:        convertTypeToSchemaType(headerType),
		Description: description,
	}

	// Handle "all" status code
	if strings.EqualFold(statusCodes, "all") {
		// Add header to all existing responses
		for code, response := range op.responses {
			if response.Headers == nil {
				response.Headers = make(map[string]routedomain.Header)
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
			response = routedomain.Response{
				Headers: make(map[string]routedomain.Header),
			}
		}

		if response.Headers == nil {
			response.Headers = make(map[string]routedomain.Header)
		}

		response.Headers[headerName] = header
		op.responses[code] = response
	}

	return nil
}

// resolveTypePathsInSchema recursively walks a domain schema tree and resolves TypePath
// for any schema that has a $ref but no TypePath yet.
func (s *Service) resolveTypePathsInSchema(schema *routedomain.Schema, file *ast.File) {
	if schema == nil {
		return
	}
	if schema.Ref != "" && schema.TypePath == "" {
		typeName := strings.TrimPrefix(schema.Ref, "#/definitions/")
		// Strip Public suffix for lookup since registry stores base types
		lookupName := typeName
		isPublicRef := strings.HasSuffix(lookupName, "Public")
		if isPublicRef {
			lookupName = strings.TrimSuffix(lookupName, "Public")
		}
		if tp := s.resolveTypePath(lookupName, file); tp != "" {
			if isPublicRef {
				schema.TypePath = tp + "Public"
			} else {
				schema.TypePath = tp
			}
		}
	}
	s.resolveTypePathsInSchema(schema.Items, file)
	s.resolveTypePathsInSchema(schema.AdditionalProperties, file)
	for _, prop := range schema.Properties {
		s.resolveTypePathsInSchema(prop, file)
	}
	for _, allOf := range schema.AllOf {
		s.resolveTypePathsInSchema(allOf, file)
	}
}

// isStructType checks if a qualified type name refers to a struct type via the registry.
// Returns true if the type is a struct or if the registry is unavailable (safe default).
func (s *Service) isStructType(qualifiedTypeName string) bool {
	if s.registry == nil {
		return true // safe default when registry unavailable
	}

	typeDef := s.registry.FindTypeSpec(qualifiedTypeName, nil)
	if typeDef == nil {
		return true // unknown type, assume struct (safe default)
	}

	_, isStruct := typeDef.TypeSpec.Type.(*ast.StructType)
	return isStruct
}

// hasNoPublicAnnotation checks if a type has @NoPublic annotation
func (s *Service) hasNoPublicAnnotation(qualifiedTypeName string) bool {
	if s.registry == nil {
		return false
	}

	// Look up the type in registry
	typeDef := s.registry.FindTypeSpec(qualifiedTypeName, nil)
	if typeDef == nil {
		return false
	}

	// Check if the type has @NoPublic in its documentation
	// The comment is attached to the GenDecl (not TypeSpec)
	if genDecl, ok := typeDef.ParentSpec.(*ast.GenDecl); ok && genDecl != nil {
		if genDecl.Doc != nil {
			for _, comment := range genDecl.Doc.List {
				if strings.Contains(comment.Text, "@NoPublic") {
					return true
				}
			}
		}
	}

	// Also check TypeSpec.Doc as a fallback
	if typeDef.TypeSpec != nil && typeDef.TypeSpec.Doc != nil {
		for _, comment := range typeDef.TypeSpec.Doc.List {
			if strings.Contains(comment.Text, "@NoPublic") {
				return true
			}
		}
	}

	return false
}
