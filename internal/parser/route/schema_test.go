package route

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTypeResolver is a mock implementation of TypeSchemaResolver for testing
type mockTypeResolver struct {
	schemas map[string]*spec.Schema
}

func newMockTypeResolver() *mockTypeResolver {
	return &mockTypeResolver{
		schemas: make(map[string]*spec.Schema),
	}
}

func (m *mockTypeResolver) GetTypeSchema(typeName string, file *ast.File, ref bool) (*spec.Schema, error) {
	if schema, ok := m.schemas[typeName]; ok {
		if ref {
			// Return a reference schema
			return &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef("#/definitions/" + typeName),
				},
			}, nil
		}
		return schema, nil
	}
	// Return nil for unknown types (primitive types should be handled differently)
	return nil, nil
}

func (m *mockTypeResolver) ParseTypeExpr(file *ast.File, typeExpr ast.Expr, ref bool) (*spec.Schema, error) {
	// Not needed for these tests
	return nil, nil
}

func (m *mockTypeResolver) addMockType(name string, schema *spec.Schema) {
	m.schemas[name] = schema
}

// TestParseParamWithModelType tests parsing parameters with model types
func TestParseParamWithModelType(t *testing.T) {
	t.Run("should parse body parameter with model type", func(t *testing.T) {
		src := `
package test

// @Param user body model.User true "User object"
// @Router /users [post]
func CreateUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		// Create mock type resolver with User model
		resolver := newMockTypeResolver()
		resolver.addMockType("model.User", &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"id": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"string"},
						},
					},
					"name": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"string"},
						},
					},
				},
			},
		})

		service := NewService(nil, "")
		service.typeResolver = resolver

		routes, err := service.ParseRoutes(astFile)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		params := routes[0].Parameters
		require.Len(t, params, 1)

		param := params[0]
		assert.Equal(t, "user", param.Name)
		assert.Equal(t, "body", param.In)
		assert.True(t, param.Required)
		assert.NotNil(t, param.Schema)

		// Should have a reference to the model
		assert.NotEmpty(t, param.Schema.Ref)
		assert.Contains(t, param.Schema.Ref, "model.User")
	})

	t.Run("should parse query parameter with model type", func(t *testing.T) {
		src := `
package test

// @Param filter query model.Filter false "Filter object"
// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		// Create mock type resolver with Filter model
		resolver := newMockTypeResolver()
		resolver.addMockType("model.Filter", &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"status": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"string"},
						},
					},
				},
			},
		})

		service := NewService(nil, "")
		service.typeResolver = resolver

		routes, err := service.ParseRoutes(astFile)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		params := routes[0].Parameters
		require.Len(t, params, 1)

		// For query parameters with objects, should expand into multiple params
		// based on the object's properties
		assert.Equal(t, "filter", params[0].Name)
		assert.Equal(t, "query", params[0].In)
	})

	t.Run("should parse array parameter with model type", func(t *testing.T) {
		src := `
package test

// @Param users body []model.User true "Array of users"
// @Router /users/bulk [post]
func CreateUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		// Create mock type resolver
		resolver := newMockTypeResolver()
		resolver.addMockType("model.User", &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
			},
		})

		service := NewService(nil, "")
		service.typeResolver = resolver

		routes, err := service.ParseRoutes(astFile)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		params := routes[0].Parameters
		require.Len(t, params, 1)

		param := params[0]
		assert.Equal(t, "users", param.Name)
		assert.Equal(t, "body", param.In)
		assert.NotNil(t, param.Schema)

		// Should be array type with items referencing model
		assert.Equal(t, "array", param.Schema.Type)
		assert.NotNil(t, param.Schema.Items)
		assert.NotEmpty(t, param.Schema.Items.Ref)
	})
}

// TestParseResponseWithModelType tests parsing responses with model types
func TestParseResponseWithModelType(t *testing.T) {
	t.Run("should parse success response with model type", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} model.User "User details"
// @Router /users/{id} [get]
func GetUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		// Create mock type resolver
		resolver := newMockTypeResolver()
		resolver.addMockType("model.User", &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"id": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"string"},
						},
					},
				},
			},
		})

		service := NewService(nil, "")
		service.typeResolver = resolver

		routes, err := service.ParseRoutes(astFile)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		response := responses[200]
		assert.Equal(t, "User details", response.Description)
		assert.NotNil(t, response.Schema)

		// Should have a reference to the model
		assert.NotEmpty(t, response.Schema.Ref)
		assert.Contains(t, response.Schema.Ref, "model.User")
	})

	t.Run("should parse success response with array of models", func(t *testing.T) {
		src := `
package test

// @Success 200 {array} model.User "List of users"
// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		// Create mock type resolver
		resolver := newMockTypeResolver()
		resolver.addMockType("model.User", &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
			},
		})

		service := NewService(nil, "")
		service.typeResolver = resolver

		routes, err := service.ParseRoutes(astFile)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		response := responses[200]
		assert.NotNil(t, response.Schema)

		// Should be array type
		assert.Equal(t, "array", response.Schema.Type)
		assert.NotNil(t, response.Schema.Items)

		// Items should reference the model
		assert.NotEmpty(t, response.Schema.Items.Ref)
		assert.Contains(t, response.Schema.Items.Ref, "model.User")
	})

	t.Run("should parse failure response with error model", func(t *testing.T) {
		src := `
package test

// @Failure 400 {object} model.ErrorResponse "Bad Request"
// @Router /users [post]
func CreateUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		// Create mock type resolver
		resolver := newMockTypeResolver()
		resolver.addMockType("model.ErrorResponse", &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"error": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"string"},
						},
					},
				},
			},
		})

		service := NewService(nil, "")
		service.typeResolver = resolver

		routes, err := service.ParseRoutes(astFile)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 400)

		response := responses[400]
		assert.NotNil(t, response.Schema)
		assert.NotEmpty(t, response.Schema.Ref)
		assert.Contains(t, response.Schema.Ref, "model.ErrorResponse")
	})
}

// TestIsModelType tests detection of model types vs primitives
func TestIsModelType(t *testing.T) {
	testCases := []struct {
		name       string
		typeName   string
		isModel    bool
	}{
		{"primitive int", "int", false},
		{"primitive string", "string", false},
		{"primitive bool", "bool", false},
		{"primitive float64", "float64", false},
		{"object keyword", "object", false},
		{"model with dot", "model.User", true},
		{"model with package", "github.com/user/api/model.Account", true},
		{"simple type", "User", true},
		{"file type", "file", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isModelType(tc.typeName)
			assert.Equal(t, tc.isModel, result, "isModelType(%s) = %v, want %v", tc.typeName, result, tc.isModel)
		})
	}
}

// TestIsModelType_ExtendedPrimitives tests that extended primitives are NOT treated as models
func TestIsModelType_ExtendedPrimitives(t *testing.T) {
	testCases := []struct {
		name     string
		typeName string
		isModel  bool
	}{
		// Extended primitives should NOT be models
		{"time.Time", "time.Time", false},
		{"*time.Time", "*time.Time", false},
		{"uuid.UUID", "uuid.UUID", false},
		{"*uuid.UUID", "*uuid.UUID", false},
		{"types.UUID", "types.UUID", false},
		{"github.com/google/uuid.UUID", "github.com/google/uuid.UUID", false},
		{"github.com/griffnb/core/lib/types.UUID", "github.com/griffnb/core/lib/types.UUID", false},
		{"decimal.Decimal", "decimal.Decimal", false},
		{"*decimal.Decimal", "*decimal.Decimal", false},
		{"github.com/shopspring/decimal.Decimal", "github.com/shopspring/decimal.Decimal", false},

		// Real models SHOULD be models
		{"model.User", "model.User", true},
		{"time.Timer", "time.Timer", true}, // Not time.Time
		{"*model.Account", "*model.Account", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isModelType(tc.typeName)
			assert.Equal(t, tc.isModel, result, "isModelType(%s) = %v, want %v", tc.typeName, result, tc.isModel)
		})
	}
}

// TestConvertTypeToSchemaType tests conversion of Go types to OpenAPI schema types
func TestConvertTypeToSchemaType(t *testing.T) {
	testCases := []struct {
		name       string
		dataType   string
		schemaType string
	}{
		// Basic Go types
		{"string", "string", "string"},
		{"int", "int", "integer"},
		{"int32", "int32", "integer"},
		{"int64", "int64", "integer"},
		{"uint", "uint", "integer"},
		{"float32", "float32", "number"},
		{"float64", "float64", "number"},
		{"bool", "bool", "boolean"},

		// Extended primitives - should have correct schema types
		{"time.Time", "time.Time", "string"},
		{"*time.Time", "*time.Time", "string"},
		{"uuid.UUID", "uuid.UUID", "string"},
		{"*uuid.UUID", "*uuid.UUID", "string"},
		{"types.UUID", "types.UUID", "string"},
		{"github.com/google/uuid.UUID", "github.com/google/uuid.UUID", "string"},
		{"decimal.Decimal", "decimal.Decimal", "number"},
		{"*decimal.Decimal", "*decimal.Decimal", "number"},
		{"github.com/shopspring/decimal.Decimal", "github.com/shopspring/decimal.Decimal", "number"},

		// Custom types should be object
		{"User", "User", "object"},
		{"model.User", "model.User", "object"},
		{"github.com/myapp/model.Account", "github.com/myapp/model.Account", "object"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertTypeToSchemaType(tc.dataType)
			assert.Equal(t, tc.schemaType, result, "convertTypeToSchemaType(%s) = %s, want %s", tc.dataType, result, tc.schemaType)
		})
	}
}
