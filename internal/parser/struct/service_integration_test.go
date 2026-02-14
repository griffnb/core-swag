package structparser_test

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/parser/field"
	structparser "github.com/griffnb/core-swag/internal/parser/struct"
)

// TestParseStructIntegration tests parsing a complete struct with multiple fields
func TestParseStructIntegration(t *testing.T) {
	src := `
package test

type User struct {
	ID        int    ` + "`json:\"id\"`" + `
	Name      string ` + "`json:\"name\" binding:\"required\"`" + `
	Email     string ` + "`json:\"email,omitempty\"`" + `
	Age       int    ` + "`json:\"age\" validate:\"min=0,max=150\"`" + `
	IsActive  bool   ` + "`json:\"is_active\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	// Extract the struct type
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType = st
				return false
			}
		}
		return true
	})

	if structType == nil {
		t.Fatal("should find struct type")
	}

	// Create service
	schemaHelper := &mockSchemaHelper{}
	config := &mockConfig{
		namingStrategy:    field.CamelCase,
		requiredByDefault: false,
	}
	typeResolver := &mockTypeResolver{}
	service := structparser.NewService(schemaHelper, config, typeResolver)

	// Parse the struct
	schema, err := service.ParseStruct(file, structType.Fields)

	// Assertions
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}
	if schema == nil {
		t.Fatal("schema should not be nil")
	}
	if len(schema.Type) == 0 || schema.Type[0] != field.OBJECT {
		t.Fatalf("schema type should be object, got %v", schema.Type)
	}

	// Check all properties are present
	expectedProps := []string{"id", "name", "email", "age", "is_active"}
	if len(schema.Properties) != len(expectedProps) {
		t.Fatalf("expected %d properties, got %d", len(expectedProps), len(schema.Properties))
	}

	for _, prop := range expectedProps {
		if _, ok := schema.Properties[prop]; !ok {
			t.Errorf("missing property: %s", prop)
		}
	}

	// Check required fields - only "name" should be required (has binding:required)
	if len(schema.Required) != 1 {
		t.Fatalf("expected 1 required field, got %d: %v", len(schema.Required), schema.Required)
	}
	if schema.Required[0] != "name" {
		t.Errorf("expected required field 'name', got '%s'", schema.Required[0])
	}

	// Verify email is not required (has omitempty)
	foundEmail := false
	for _, req := range schema.Required {
		if req == "email" {
			foundEmail = true
		}
	}
	if foundEmail {
		t.Error("email should not be required (has omitempty)")
	}
}

// TestParseStructWithEmbedded tests parsing struct with embedded fields
func TestParseStructWithEmbedded(t *testing.T) {
	src := `
package test

type Base struct {
	ID int ` + "`json:\"id\"`" + `
}

type User struct {
	Base
	Name string ` + "`json:\"name\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	// Find User struct
	var userStruct *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if ts.Name.Name == "User" {
				if st, ok := ts.Type.(*ast.StructType); ok {
					userStruct = st
					return false
				}
			}
		}
		return true
	})

	if userStruct == nil {
		t.Fatal("should find User struct")
	}

	// Create service with resolver that returns Base schema
	schemaHelper := &mockSchemaHelper{}
	config := &mockConfig{
		namingStrategy:    field.CamelCase,
		requiredByDefault: false,
	}
	typeResolver := &mockTypeResolverWithBase{}
	service := structparser.NewService(schemaHelper, config, typeResolver)

	// Parse User struct
	schema, err := service.ParseStruct(file, userStruct.Fields)

	// Assertions
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	// Should have both ID (from Base) and Name properties
	if len(schema.Properties) < 2 {
		t.Fatalf("expected at least 2 properties (id from Base, name from User), got %d", len(schema.Properties))
	}

	// Check that both fields are present
	if _, ok := schema.Properties["id"]; !ok {
		t.Error("missing 'id' property from embedded Base")
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("missing 'name' property from User")
	}
}

// mockTypeResolverWithBase returns a mock Base schema for embedded struct tests
// It implements domain.TypeSchemaResolver
type mockTypeResolverWithBase struct{}

func (m *mockTypeResolverWithBase) GetTypeSchema(typeName string, file *ast.File, ref bool) (*spec.Schema, error) {
	schema := &spec.Schema{}

	// Return a mock Base schema with ID property
	if typeName == "Base" {
		schema.Type = []string{field.OBJECT}
		schema.Properties = map[string]spec.Schema{
			"id": {
				SchemaProps: spec.SchemaProps{
					Type: []string{field.INTEGER},
				},
			},
		}
		return schema, nil
	}

	// Default behavior for other types
	switch typeName {
	case "string":
		schema.Type = []string{field.STRING}
	case "int", "int64", "int32":
		schema.Type = []string{field.INTEGER}
	case "bool":
		schema.Type = []string{field.BOOLEAN}
	default:
		schema.Ref = spec.MustCreateRef("#/definitions/" + typeName)
	}
	return schema, nil
}

func (m *mockTypeResolverWithBase) ParseTypeExpr(file *ast.File, typeExpr ast.Expr, ref bool) (*spec.Schema, error) {
	schema := &spec.Schema{}
	schema.Type = []string{field.STRING}
	return schema, nil
}
