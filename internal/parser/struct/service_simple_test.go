package structparser_test

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/swaggo/swag/internal/parser/field"
	structparser "github.com/swaggo/swag/internal/parser/struct"
)

// mockSchemaHelper implements field.SchemaHelper for testing
type mockSchemaHelper struct{}

func (m *mockSchemaHelper) BuildCustomSchema(types []string) (*spec.Schema, error) {
	return nil, nil
}

func (m *mockSchemaHelper) IsRefSchema(schema *spec.Schema) bool {
	return schema != nil && schema.Ref.String() != ""
}

func (m *mockSchemaHelper) DefineType(schemaType string, value string) (interface{}, error) {
	switch schemaType {
	case field.INTEGER:
		return 123, nil
	case field.STRING:
		return value, nil
	case field.BOOLEAN:
		return true, nil
	case field.NUMBER:
		return 1.23, nil
	default:
		return value, nil
	}
}

func (m *mockSchemaHelper) DefineTypeOfExample(schemaType, arrayType, exampleValue string) (interface{}, error) {
	return exampleValue, nil
}

func (m *mockSchemaHelper) PrimitiveSchema(refType string) *spec.Schema {
	schema := &spec.Schema{}
	schema.Type = []string{refType}
	return schema
}

func (m *mockSchemaHelper) SetExtensionParam(attr string) spec.Extensions {
	return spec.Extensions{}
}

func (m *mockSchemaHelper) GetSchemaTypePath(schema *spec.Schema, depth int) []string {
	if schema == nil || len(schema.Type) == 0 {
		return []string{}
	}
	return schema.Type
}

func (m *mockSchemaHelper) IsNumericType(typeName string) bool {
	return typeName == field.INTEGER || typeName == field.NUMBER
}

// mockConfig implements field.ParserConfig for testing
type mockConfig struct {
	namingStrategy    string
	requiredByDefault bool
}

func (m *mockConfig) GetNamingStrategy() string {
	return m.namingStrategy
}

func (m *mockConfig) IsRequiredByDefault() bool {
	return m.requiredByDefault
}

// mockTypeResolver implements domain.TypeSchemaResolver for testing
type mockTypeResolver struct{}

func (m *mockTypeResolver) GetTypeSchema(typeName string, file *ast.File, ref bool) (*spec.Schema, error) {
	// Return a simple schema based on type name
	schema := &spec.Schema{}
	switch typeName {
	case "string":
		schema.Type = []string{field.STRING}
	case "int", "int64", "int32":
		schema.Type = []string{field.INTEGER}
	case "bool":
		schema.Type = []string{field.BOOLEAN}
	case "float64", "float32":
		schema.Type = []string{field.NUMBER}
	default:
		// For unknown types, treat as object reference
		schema.Ref = spec.MustCreateRef("#/definitions/" + typeName)
	}
	return schema, nil
}

func (m *mockTypeResolver) ParseTypeExpr(file *ast.File, typeExpr ast.Expr, ref bool) (*spec.Schema, error) {
	// For testing, return a simple string schema
	schema := &spec.Schema{}
	schema.Type = []string{field.STRING}
	return schema, nil
}

// TestParseFieldBasic tests basic field parsing
func TestParseFieldBasic(t *testing.T) {
	src := `
package test

type User struct {
	Name string ` + "`json:\"name\"`" + `
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
	if len(structType.Fields.List) != 1 {
		t.Fatalf("should have 1 field, got %d", len(structType.Fields.List))
	}

	// Create service
	schemaHelper := &mockSchemaHelper{}
	config := &mockConfig{
		namingStrategy:    field.CamelCase,
		requiredByDefault: false,
	}
	typeResolver := &mockTypeResolver{}
	service := structparser.NewService(schemaHelper, config, typeResolver)

	// Parse the field
	testField := structType.Fields.List[0]
	properties, required, err := service.ParseField(file, testField)

	// Assertions
	if err != nil {
		t.Fatalf("ParseField failed: %v", err)
	}
	if properties == nil {
		t.Fatal("properties should not be nil")
	}
	if len(properties) != 1 {
		t.Fatalf("should have 1 property, got %d", len(properties))
	}
	if len(required) != 0 {
		t.Fatalf("should have no required fields (no binding:required tag), got %d", len(required))
	}

	// Check property name is "name" from json tag
	_, hasName := properties["name"]
	if !hasName {
		t.Fatal("should have 'name' property from json tag")
	}
}
