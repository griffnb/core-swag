package structparser

import (
	"go/ast"

	"github.com/go-openapi/spec"
)

// Service handles struct parsing for OpenAPI schema generation.
// It supports both standard Go structs and custom model structs with fields.StructField[T].
type Service struct {
	// TODO: Add dependencies as we extract functionality
	// registry          *registry.Service
	// schemaBuilder     *schema.BuilderService
	// fieldParserFactory FieldParserFactory
	// propNamingStrategy string
	// requiredByDefault  bool
	// overrides          map[string]string
	// structStack        []*domain.TypeSpecDef
	// parsedSchemas      map[*domain.TypeSpecDef]*spec.Schema
	// debug              Debugger
}

// NewService creates a new struct parser service
func NewService() *Service {
	return &Service{}
}

// ParseStruct parses a struct and returns its OpenAPI schema
func (s *Service) ParseStruct(file *ast.File, fields *ast.FieldList) (*spec.Schema, error) {
	// TODO: Will implement in future phases
	return nil, nil
}

// ParseField parses an individual struct field
func (s *Service) ParseField(file *ast.File, field *ast.Field) (map[string]spec.Schema, []string, error) {
	// TODO: Will implement in future phases
	return nil, nil, nil
}

// ParseDefinition parses a type definition and generates schema(s)
func (s *Service) ParseDefinition(typeSpec ast.Expr) (*spec.Schema, error) {
	// TODO: Will implement in future phases
	// This will handle both standard structs and custom models
	return nil, nil
}
