package domain

import (
	"go/ast"

	"github.com/go-openapi/spec"
)

// TypeSchemaResolver resolves Go types to OpenAPI schemas.
// This interface allows struct parsers to resolve types without depending on the main Parser.
type TypeSchemaResolver interface {
	// GetTypeSchema resolves a named type to its OpenAPI schema
	GetTypeSchema(typeName string, file *ast.File, ref bool) (*spec.Schema, error)

	// ParseTypeExpr parses an AST type expression into an OpenAPI schema
	ParseTypeExpr(file *ast.File, typeExpr ast.Expr, ref bool) (*spec.Schema, error)
}
