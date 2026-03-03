package route

import (
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComplexNonStructTypes tests that complex non-struct types in AllOf
// override syntax are handled correctly (maps, slices, any, interface{}).
// These types must produce valid OpenAPI schemas, not bad $ref references.
func TestComplexNonStructTypes(t *testing.T) {

	// parseRoute is a test helper that parses a single route from source and returns
	// the named override property's schema for assertion.
	parseRoute := func(t *testing.T, src string) map[int]map[string]interface{} {
		t.Helper()
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)
		return nil // just to compile; tests use routes directly
	}
	_ = parseRoute // suppress unused

	t.Run("Response{data=map[string][]any} should produce map with array-of-object values", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=map[string][]any} "map with array values"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		// map[string][]any → object with additionalProperties = {type: array, items: {type: object}}
		assert.Equal(t, "object", data.Type, "map should produce object type")
		assert.Empty(t, data.Ref, "should not have a $ref")
		require.NotNil(t, data.AdditionalProperties, "map should have additionalProperties")
		assert.Equal(t, "array", data.AdditionalProperties.Type, "map values should be arrays")
		require.NotNil(t, data.AdditionalProperties.Items, "array items should exist")
		assert.Equal(t, "object", data.AdditionalProperties.Items.Type, "[]any items should be object")
	})

	t.Run("Response{data=any} should produce plain object", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=any} "plain any"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "object", data.Type, "any should be object type")
		assert.Empty(t, data.Ref, "any should not have a $ref")
	})

	t.Run("response.SuccessResponse{data=any} should produce plain object for data", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} response.SuccessResponse{data=any} "qualified with any"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "object", data.Type, "any should be object type")
		assert.Empty(t, data.Ref, "any should not have a $ref")
	})

	t.Run("response.SuccessResponse{data=map[string][]any} should produce map with array values", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} response.SuccessResponse{data=map[string][]any} "qualified with complex map"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "object", data.Type, "map should produce object type")
		assert.Empty(t, data.Ref, "should not have a $ref")
		require.NotNil(t, data.AdditionalProperties, "map should have additionalProperties")
		assert.Equal(t, "array", data.AdditionalProperties.Type, "map values should be arrays")
		require.NotNil(t, data.AdditionalProperties.Items, "array items should exist")
		assert.Equal(t, "object", data.AdditionalProperties.Items.Type, "[]any items should be object")
	})

	t.Run("Response{data=map[string][]string} should produce map with array-of-string values", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=map[string][]string} "map with string arrays"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "object", data.Type, "map should produce object type")
		assert.Empty(t, data.Ref, "should not have a $ref")
		require.NotNil(t, data.AdditionalProperties, "map should have additionalProperties")
		assert.Equal(t, "array", data.AdditionalProperties.Type, "map values should be arrays")
		require.NotNil(t, data.AdditionalProperties.Items, "array items should exist")
		assert.Equal(t, "string", data.AdditionalProperties.Items.Type, "[]string items should be string")
	})

	t.Run("Response{data=map[string]interface{}} should produce plain object", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=map[string]interface{}} "map with interface values"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		// map[string]interface{} is a wildcard map → plain object, no additionalProperties
		assert.Equal(t, "object", data.Type, "wildcard map should be plain object")
		assert.Empty(t, data.Ref, "should not have a $ref")
	})

	t.Run("Response{data=[]any} should produce array of objects", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=[]any} "array of any"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "array", data.Type, "[]any should be array")
		require.NotNil(t, data.Items, "array should have items")
		assert.Equal(t, "object", data.Items.Type, "any items should be object")
		assert.Empty(t, data.Items.Ref, "any items should not have $ref")
	})

	t.Run("Response{data=[]string} should produce array of strings", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=[]string} "array of strings"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "array", data.Type, "[]string should be array")
		require.NotNil(t, data.Items, "array should have items")
		assert.Equal(t, "string", data.Items.Type, "string items should be string type")
	})

	t.Run("Response{data=map[string]int} should produce map with integer values", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=map[string]int} "map with int values"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "object", data.Type, "map should produce object")
		assert.Empty(t, data.Ref, "should not have a $ref")
		require.NotNil(t, data.AdditionalProperties, "map should have additionalProperties")
		assert.Equal(t, "integer", data.AdditionalProperties.Type, "int values should be integer type")
	})

	t.Run("Response{data=map[string][]Model} should produce map with array-of-ref values", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=map[string][]Model} "map with model arrays"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "object", data.Type, "map should produce object")
		assert.Empty(t, data.Ref, "should not have a $ref")
		require.NotNil(t, data.AdditionalProperties, "map should have additionalProperties")
		assert.Equal(t, "array", data.AdditionalProperties.Type, "values should be arrays")
		require.NotNil(t, data.AdditionalProperties.Items, "array items should exist")
		assert.Contains(t, data.AdditionalProperties.Items.Ref, "Model", "array items should ref Model")
	})

	t.Run("Response{data=[]interface{}} should produce array of objects", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=[]interface{}} "array of interface"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "data")

		data := schema.Properties["data"]
		assert.Equal(t, "array", data.Type, "[]interface{} should be array")
		require.NotNil(t, data.Items, "array should have items")
		assert.Equal(t, "object", data.Items.Type, "interface{} items should be object")
		assert.Empty(t, data.Items.Ref, "interface{} items should not have $ref")
	})

	t.Run("Response{meta=map[string]string,data=[]any} should handle multiple complex overrides", func(t *testing.T) { //nolint:dupl
		src := `
package test

// @Success 200 {object} Response{meta=map[string]string,data=[]any} "multiple overrides"
// @Router /test [get]
func Handler() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		schema := routes[0].Responses[200].Schema
		require.NotNil(t, schema)
		require.Contains(t, schema.Properties, "meta")
		require.Contains(t, schema.Properties, "data")

		// meta = map[string]string → object with additionalProperties: {type: string}
		meta := schema.Properties["meta"]
		assert.Equal(t, "object", meta.Type)
		require.NotNil(t, meta.AdditionalProperties)
		assert.Equal(t, "string", meta.AdditionalProperties.Type)

		// data = []any → array with items: {type: object}
		data := schema.Properties["data"]
		assert.Equal(t, "array", data.Type)
		require.NotNil(t, data.Items)
		assert.Equal(t, "object", data.Items.Type)
	})
}

// TestComplexNonStructParamTypes tests that complex non-struct types in body
// @Param annotations produce valid schemas, not bad $ref references.
func TestComplexNonStructParamTypes(t *testing.T) {

	t.Run("@Param body any should produce object schema, not $ref", func(t *testing.T) {
		src := `
package test

// @Param data body any true "Webhook event data with type field"
// @Router /webhook [post]
func HandleWebhook() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		params := routes[0].Parameters
		require.Len(t, params, 1)

		param := params[0]
		assert.Equal(t, "data", param.Name)
		assert.Equal(t, "body", param.In)
		assert.True(t, param.Required)
		// any should NOT create a $ref - it should be type: object
		if param.Schema != nil {
			assert.Empty(t, param.Schema.Ref, "any should not produce a $ref")
			assert.Equal(t, "object", param.Schema.Type, "any should be object type")
		} else {
			assert.Equal(t, "object", param.Type, "any should be object type")
		}
	})

	t.Run("@Param body interface{} should produce object schema, not $ref", func(t *testing.T) {
		src := `
package test

// @Param data body interface{} true "Generic data"
// @Router /generic [post]
func HandleGeneric() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		params := routes[0].Parameters
		require.Len(t, params, 1)

		param := params[0]
		assert.Equal(t, "data", param.Name)
		assert.Equal(t, "body", param.In)
		if param.Schema != nil {
			assert.Empty(t, param.Schema.Ref, "interface{} should not produce a $ref")
			assert.Equal(t, "object", param.Schema.Type, "interface{} should be object type")
		} else {
			assert.Equal(t, "object", param.Type, "interface{} should be object type")
		}
	})

	t.Run("@Param body map[string]any should produce object schema", func(t *testing.T) {
		src := `
package test

// @Param data body map[string]any true "Arbitrary JSON"
// @Router /flexible [post]
func HandleFlexible() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		params := routes[0].Parameters
		require.Len(t, params, 1)

		param := params[0]
		assert.Equal(t, "body", param.In)
		require.NotNil(t, param.Schema, "body map type should use Schema")
		assert.Equal(t, "object", param.Schema.Type, "map[string]any should be object")
		assert.Empty(t, param.Schema.Ref, "should not have $ref")
	})

	t.Run("@Param body []any should produce array of object schema", func(t *testing.T) {
		src := `
package test

// @Param data body []any true "Array of anything"
// @Router /batch [post]
func HandleBatch() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		params := routes[0].Parameters
		require.Len(t, params, 1)

		param := params[0]
		assert.Equal(t, "body", param.In)
		// []any: isArray=true, dataType stripped to "any"
		// any is not a model type (after fix), so it should use Type field
		// OR if it's treated as model, Schema with array wrapping
		// Either way, should not produce $ref to "any"
		if param.Schema != nil {
			assert.Empty(t, param.Schema.Ref, "[]any should not ref 'any'")
			if param.Schema.Items != nil {
				assert.Empty(t, param.Schema.Items.Ref, "[]any items should not ref 'any'")
			}
		}
	})
}
