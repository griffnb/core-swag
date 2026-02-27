package route

import (
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseOperation tests parsing operation from function declaration
func TestParseOperation(t *testing.T) {
	t.Run("should parse basic operation from comments", func(t *testing.T) {
		src := `
package test

// GetUser returns a user
// @Summary Get a user
// @Description Get user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} string "Success"
// @Router /users/{id} [get]
func GetUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		// Create service
		service := NewService(nil, "")
		require.NotNil(t, service)

		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		route := routes[0]
		assert.Equal(t, "GET", route.Method)
		assert.Equal(t, "/users/{id}", route.Path)
		assert.Equal(t, "Get a user", route.Summary)
		assert.Equal(t, "Get user by ID", route.Description)
		assert.Equal(t, []string{"users"}, route.Tags)
	})

	t.Run("should parse multiple routes from one function", func(t *testing.T) {
		src := `
package test

// GetOrCreateUser handles GET and POST
// @Summary Get or create user
// @Router /users [get]
// @Router /users [post]
func GetOrCreateUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 2)

		assert.Equal(t, "GET", routes[0].Method)
		assert.Equal(t, "/users", routes[0].Path)
		assert.Equal(t, "POST", routes[1].Method)
		assert.Equal(t, "/users", routes[1].Path)
	})

	t.Run("should skip functions without router annotation", func(t *testing.T) {
		src := `
package test

// Helper is a helper function
func Helper() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		assert.Empty(t, routes)
	})
}

// TestParseRouter tests @router annotation parsing
func TestParseRouter(t *testing.T) {
	t.Run("should parse simple router annotation", func(t *testing.T) {
		src := `
package test

// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		assert.Equal(t, "GET", routes[0].Method)
		assert.Equal(t, "/users", routes[0].Path)
	})

	t.Run("should parse router with path parameters", func(t *testing.T) {
		src := `
package test

// @Router /users/{id} [get]
func GetUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		assert.Equal(t, "/users/{id}", routes[0].Path)
	})

	t.Run("should parse deprecated router", func(t *testing.T) {
		src := `
package test

// @DeprecatedRouter /old-path [get]
func OldEndpoint() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		assert.True(t, routes[0].Deprecated)
	})

	t.Run("should handle complex paths", func(t *testing.T) {
		testCases := []struct {
			name     string
			path     string
			expected string
		}{
			{"with plus sign", "/path/+action", "/path/+action"},
			{"with dollar sign", "/path/$id", "/path/$id"},
			{"with parens", "/path/(optional)", "/path/(optional)"},
			{"with colon", "/path/:id", "/path/:id"},
			{"with tilde", "/path/~user", "/path/~user"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				src := `
package test

// @Router ` + tc.path + ` [get]
func Handler() {}
`
				fset := token.NewFileSet()
				astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
				require.NoError(t, err)

				service := NewService(nil, "")
				routes, err := service.ParseRoutes(astFile, "test.go", fset)
				require.NoError(t, err)
				require.Len(t, routes, 1)

				assert.Equal(t, tc.expected, routes[0].Path)
			})
		}
	})
}

// TestParseParam tests @param annotation parsing
func TestParseParam(t *testing.T) {
	t.Run("should parse path parameter", func(t *testing.T) {
		src := `
package test

// @Param id path int true "User ID"
// @Router /users/{id} [get]
func GetUser() {}
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

		assert.Equal(t, "id", params[0].Name)
		assert.Equal(t, "path", params[0].In)
		assert.Equal(t, "integer", params[0].Type)
		assert.True(t, params[0].Required)
		assert.Equal(t, "User ID", params[0].Description)
	})

	t.Run("should parse query parameter", func(t *testing.T) {
		src := `
package test

// @Param limit query int false "Limit results"
// @Router /users [get]
func GetUsers() {}
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

		assert.Equal(t, "limit", params[0].Name)
		assert.Equal(t, "query", params[0].In)
		assert.False(t, params[0].Required)
	})

	t.Run("should parse array parameter", func(t *testing.T) {
		src := `
package test

// @Param ids query []int true "User IDs"
// @Router /users [get]
func GetUsers() {}
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

		assert.Equal(t, "array", params[0].Type)
		assert.Equal(t, "integer", params[0].Items.Type)
	})

	t.Run("should parse body parameter", func(t *testing.T) {
		src := `
package test

// @Param user body object true "User object"
// @Router /users [post]
func CreateUser() {}
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

		assert.Equal(t, "user", params[0].Name)
		assert.Equal(t, "body", params[0].In)
	})
}

// TestParseSuccess tests @success annotation parsing
func TestParseSuccess(t *testing.T) {
	t.Run("should parse success response with description only", func(t *testing.T) {
		src := `
package test

// @Success 200 "OK"
// @Router /health [get]
func Health() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		assert.Equal(t, "OK", responses[200].Description)
	})

	t.Run("should parse success response with schema", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} string "Success"
// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		assert.Equal(t, "Success", responses[200].Description)
		assert.NotNil(t, responses[200].Schema)
	})

	t.Run("should parse success response with array schema", func(t *testing.T) {
		src := `
package test

// @Success 200 {array} string "List of users"
// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		assert.NotNil(t, responses[200].Schema)
	})

	t.Run("should parse multiple status codes", func(t *testing.T) {
		src := `
package test

// @Success 200,201,204 "Success"
// @Router /users [post]
func CreateUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		assert.Contains(t, responses, 200)
		assert.Contains(t, responses, 201)
		assert.Contains(t, responses, 204)
	})
}

// TestParseFailure tests @failure annotation parsing
func TestParseFailure(t *testing.T) {
	t.Run("should parse failure response", func(t *testing.T) {
		src := `
package test

// @Failure 400 "Bad Request"
// @Router /users [post]
func CreateUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 400)

		assert.Equal(t, "Bad Request", responses[400].Description)
	})

	t.Run("should parse failure with schema", func(t *testing.T) {
		src := `
package test

// @Failure 500 {object} string "Internal Server Error"
// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 500)

		assert.NotNil(t, responses[500].Schema)
	})
}

// TestParsePublic tests @public annotation parsing
func TestParsePublic(t *testing.T) {
	t.Run("should mark route as public", func(t *testing.T) {
		src := `
package test

// @Public
// @Router /login [post]
func Login() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		assert.True(t, routes[0].IsPublic)
	})

	t.Run("should default to non-public", func(t *testing.T) {
		src := `
package test

// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		assert.False(t, routes[0].IsPublic)
	})
}

// TestParseHeader tests @header annotation parsing
func TestParseHeader(t *testing.T) {
	t.Run("should parse response header", func(t *testing.T) {
		src := `
package test

// @Success 200 "OK"
// @Header 200 {string} X-Request-Id "Request ID"
// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		headers := responses[200].Headers
		require.Contains(t, headers, "X-Request-Id")

		assert.Equal(t, "string", headers["X-Request-Id"].Type)
		assert.Equal(t, "Request ID", headers["X-Request-Id"].Description)
	})

	t.Run("should parse header for all responses", func(t *testing.T) {
		src := `
package test

// @Success 200 "OK"
// @Failure 400 "Bad Request"
// @Header all {string} X-Request-Id "Request ID"
// @Router /users [post]
func CreateUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		responses := routes[0].Responses
		require.Contains(t, responses, 200)
		require.Contains(t, responses, 400)

		// Both responses should have the header
		assert.Contains(t, responses[200].Headers, "X-Request-Id")
		assert.Contains(t, responses[400].Headers, "X-Request-Id")
	})
}

// TestParseSecurityComment tests security annotation parsing
func TestParseSecurityComment(t *testing.T) {
	t.Run("should parse simple security", func(t *testing.T) {
		src := `
package test

// @Security ApiKeyAuth
// @Router /users [get]
func GetUsers() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		assert.NotEmpty(t, routes[0].Security)
	})

	t.Run("should parse security with scopes", func(t *testing.T) {
		src := `
package test

// @Security OAuth2[read, write]
// @Router /users [post]
func CreateUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		assert.NotEmpty(t, routes[0].Security)
	})
}

// TestCompleteOperationParsing tests parsing a complete operation with all annotations
func TestCompleteOperationParsing(t *testing.T) {
	src := `
package test

// GetUser retrieves a user by ID
// @Summary Get a user
// @Description Get detailed user information by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param limit query int false "Limit results"
// @Success 200 {object} string "User details"
// @Failure 400 {object} string "Bad Request"
// @Failure 404 {object} string "Not Found"
// @Header 200 {string} X-Request-Id "Request ID"
// @Security ApiKeyAuth
// @Router /users/{id} [get]
func GetUser() {}
`
	fset := token.NewFileSet()
	astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
	require.NoError(t, err)

	service := NewService(nil, "")
	routes, err := service.ParseRoutes(astFile, "test.go", fset)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	route := routes[0]

	// Basic info
	assert.Equal(t, "GET", route.Method)
	assert.Equal(t, "/users/{id}", route.Path)
	assert.Equal(t, "Get a user", route.Summary)
	assert.Equal(t, "Get detailed user information by ID", route.Description)
	assert.Equal(t, []string{"users"}, route.Tags)

	// Parameters
	require.Len(t, route.Parameters, 2)
	assert.Equal(t, "id", route.Parameters[0].Name)
	assert.Equal(t, "limit", route.Parameters[1].Name)

	// Responses
	assert.Contains(t, route.Responses, 200)
	assert.Contains(t, route.Responses, 400)
	assert.Contains(t, route.Responses, 404)

	// Headers
	assert.Contains(t, route.Responses[200].Headers, "X-Request-Id")

	// Security
	assert.NotEmpty(t, route.Security)

	// Content types
	assert.Contains(t, route.Consumes, "application/json")
	assert.Contains(t, route.Produces, "application/json")
}

// TestPublicAnnotationWithResponses tests @Public annotation affects response schema refs
func TestPublicAnnotationWithResponses(t *testing.T) {
	t.Run("should use Public variant for model references when @Public is set", func(t *testing.T) {
		src := `
package test

// @Public
// @Success 200 {object} Account "Success"
// @Router /auth/me [get]
func GetMe() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		route := routes[0]
		assert.True(t, route.IsPublic)

		// Check response schema uses AccountPublic
		responses := route.Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)
		assert.Contains(t, schema.Ref, "AccountPublic")
		assert.NotContains(t, schema.Ref, "Account\"")
	})

	t.Run("should use Public variant for array responses", func(t *testing.T) {
		src := `
package test

// @Public
// @Success 200 {array} Account "List of accounts"
// @Router /public/accounts [get]
func GetAccounts() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		route := routes[0]
		assert.True(t, route.IsPublic)

		// Check array items use AccountPublic
		responses := route.Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)
		assert.Equal(t, "array", schema.Type)
		require.NotNil(t, schema.Items)
		assert.Contains(t, schema.Items.Ref, "AccountPublic")
	})

	t.Run("should use regular model when @Public is not set", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Account "Success"
// @Router /internal/account [get]
func GetAccount() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		route := routes[0]
		assert.False(t, route.IsPublic)

		// Check response schema uses Account (not AccountPublic)
		responses := route.Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)
		assert.Contains(t, schema.Ref, "Account")
		assert.NotContains(t, schema.Ref, "Public")
	})

	t.Run("should handle qualified types with @Public", func(t *testing.T) {
		src := `
package test

// @Public
// @Success 200 {object} model.User "User details"
// @Router /public/user [get]
func GetUser() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		route := routes[0]
		assert.True(t, route.IsPublic)

		// Check response schema uses model.UserPublic
		responses := route.Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)
		assert.Contains(t, schema.Ref, "model.UserPublic")
	})

	t.Run("should not affect primitive types", func(t *testing.T) {
		src := `
package test

// @Public
// @Success 200 {object} string "Success message"
// @Router /public/status [get]
func GetStatus() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		route := routes[0]
		assert.True(t, route.IsPublic)

		// Check response schema is still string type (primitives not affected)
		responses := route.Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)
		assert.Equal(t, "string", schema.Type)
		assert.Empty(t, schema.Ref) // No reference for primitives
	})
}

// TestAllOfComposition tests AllOf composition for combined types
func TestAllOfComposition(t *testing.T) {
	t.Run("should build AllOf for combined type Response{data=Account}", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=Account} "Success with account data"
// @Router /account [get]
func GetAccount() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		// Check response schema uses AllOf composition
		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)

		// AllOf creates an object with properties
		// The base Response is referenced, and data field is overridden
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Properties, "data")
	})

	t.Run("should build AllOf with array field Response{data=[]Account}", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=[]Account} "Success with account list"
// @Router /accounts [get]
func GetAccounts() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		// Check response schema
		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)

		// Should have data property with array type
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Properties, "data")

		dataSchema := schema.Properties["data"]
		assert.Equal(t, "array", dataSchema.Type)
		assert.NotNil(t, dataSchema.Items)
	})

	t.Run("should build AllOf with multiple fields Response{data=Account,meta=Meta}", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{data=Account,meta=Meta} "Success with account and metadata"
// @Router /account/detail [get]
func GetAccountDetail() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		// Check response schema has both overrides
		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)

		// Should have both data and meta properties
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Properties, "data")
		assert.Contains(t, schema.Properties, "meta")
	})

	t.Run("should build AllOf with @Public annotation Response{data=Account}", func(t *testing.T) {
		src := `
package test

// @Public
// @Success 200 {object} Response{data=Account} "Success with public account"
// @Router /public/account [get]
func GetPublicAccount() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		route := routes[0]
		assert.True(t, route.IsPublic)

		// Check response schema uses Public variants
		responses := route.Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)

		// Base type should be ResponsePublic (if Response has Public variant)
		// Or just Response if it doesn't
		// Data field should reference AccountPublic
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Properties, "data")

		dataSchema := schema.Properties["data"]
		assert.Contains(t, dataSchema.Ref, "AccountPublic")
	})

	t.Run("should build AllOf with qualified types Response{data=model.Account}", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} response.SuccessResponse{data=model.Account} "Success"
// @Router /api/account [get]
func GetAPIAccount() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		// Check response schema handles qualified types
		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)

		// Should reference qualified types
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Properties, "data")

		dataSchema := schema.Properties["data"]
		assert.Contains(t, dataSchema.Ref, "model.Account")
	})

	t.Run("should handle primitive field overrides Response{count=int}", func(t *testing.T) {
		src := `
package test

// @Success 200 {object} Response{count=int} "Success with count"
// @Router /count [get]
func GetCount() {}
`
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		require.NoError(t, err)

		service := NewService(nil, "")
		routes, err := service.ParseRoutes(astFile, "test.go", fset)
		require.NoError(t, err)
		require.Len(t, routes, 1)

		// Check response schema handles primitive overrides
		responses := routes[0].Responses
		require.Contains(t, responses, 200)

		schema := responses[200].Schema
		require.NotNil(t, schema)

		// Should have count property with integer type
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Properties, "count")

		countSchema := schema.Properties["count"]
		assert.Equal(t, "integer", countSchema.Type)
	})
}
