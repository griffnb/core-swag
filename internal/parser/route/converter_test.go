package route

import (
	"testing"

	"github.com/griffnb/core-swag/internal/parser/route/domain"
)

func TestRouteToSpecOperation(t *testing.T) {
	t.Run("nil route returns nil", func(t *testing.T) {
		result := RouteToSpecOperation(nil)
		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})

	t.Run("converts basic route", func(t *testing.T) {
		route := &domain.Route{
			Method:       "GET",
			Path:         "/users/{id}",
			Summary:      "Get user by ID",
			Description:  "Returns a user by ID",
			Tags:         []string{"users"},
			OperationID:  "getUser",
			FilePath:     "/path/to/file.go",
			FunctionName: "GetUser",
			LineNumber:   42,
		}

		result := RouteToSpecOperation(route)

		if result.Summary != "Get user by ID" {
			t.Errorf("Expected summary 'Get user by ID', got %s", result.Summary)
		}
		if result.Description != "Returns a user by ID" {
			t.Errorf("Expected description 'Returns a user by ID', got %s", result.Description)
		}
		if result.ID != "getUser" {
			t.Errorf("Expected ID 'getUser', got %s", result.ID)
		}
		if len(result.Tags) != 1 || result.Tags[0] != "users" {
			t.Errorf("Expected tags ['users'], got %v", result.Tags)
		}
	})

	t.Run("adds source location extensions", func(t *testing.T) {
		route := &domain.Route{
			FilePath:     "/src/handlers/user.go",
			FunctionName: "GetUserHandler",
			LineNumber:   123,
		}

		result := RouteToSpecOperation(route)

		if result.Extensions["x-path"] != "/src/handlers/user.go" {
			t.Errorf("Expected x-path '/src/handlers/user.go', got %v", result.Extensions["x-path"])
		}
		if result.Extensions["x-function"] != "GetUserHandler" {
			t.Errorf("Expected x-function 'GetUserHandler', got %v", result.Extensions["x-function"])
		}
		if result.Extensions["x-line"] != 123 {
			t.Errorf("Expected x-line 123, got %v", result.Extensions["x-line"])
		}
	})

	t.Run("converts parameters", func(t *testing.T) {
		route := &domain.Route{
			Parameters: []domain.Parameter{
				{
					Name:        "id",
					In:          "path",
					Type:        "integer",
					Required:    true,
					Description: "User ID",
					Format:      "int64",
				},
				{
					Name:        "name",
					In:          "query",
					Type:        "string",
					Required:    false,
					Description: "User name filter",
				},
			},
		}

		result := RouteToSpecOperation(route)

		if len(result.Parameters) != 2 {
			t.Fatalf("Expected 2 parameters, got %d", len(result.Parameters))
		}

		param0 := result.Parameters[0]
		if param0.Name != "id" {
			t.Errorf("Expected param name 'id', got %s", param0.Name)
		}
		if param0.In != "path" {
			t.Errorf("Expected param in 'path', got %s", param0.In)
		}
		if param0.Type != "integer" {
			t.Errorf("Expected param type 'integer', got %s", param0.Type)
		}
		if !param0.Required {
			t.Error("Expected param to be required")
		}
	})

	t.Run("converts responses", func(t *testing.T) {
		route := &domain.Route{
			Responses: map[int]domain.Response{
				200: {
					Description: "Success",
					Schema: &domain.Schema{
						Type: "object",
						Ref:  "#/definitions/User",
					},
				},
				404: {
					Description: "Not Found",
				},
			},
		}

		result := RouteToSpecOperation(route)

		if result.Responses == nil {
			t.Fatal("Expected responses to be set")
		}

		if len(result.Responses.StatusCodeResponses) != 2 {
			t.Fatalf("Expected 2 responses, got %d", len(result.Responses.StatusCodeResponses))
		}

		resp200 := result.Responses.StatusCodeResponses[200]
		if resp200.Description != "Success" {
			t.Errorf("Expected response 200 description 'Success', got %s", resp200.Description)
		}

		resp404 := result.Responses.StatusCodeResponses[404]
		if resp404.Description != "Not Found" {
			t.Errorf("Expected response 404 description 'Not Found', got %s", resp404.Description)
		}
	})

	t.Run("converts security", func(t *testing.T) {
		route := &domain.Route{
			Security: []map[string][]string{
				{
					"api_key": {},
				},
				{
					"oauth2": {"read", "write"},
				},
			},
		}

		result := RouteToSpecOperation(route)

		if len(result.Security) != 2 {
			t.Fatalf("Expected 2 security requirements, got %d", len(result.Security))
		}

		if _, ok := result.Security[0]["api_key"]; !ok {
			t.Error("Expected api_key security requirement")
		}

		if scopes, ok := result.Security[1]["oauth2"]; !ok || len(scopes) != 2 {
			t.Errorf("Expected oauth2 security with 2 scopes, got %v", result.Security[1])
		}
	})

	t.Run("converts consumes and produces", func(t *testing.T) {
		route := &domain.Route{
			Consumes: []string{"application/json", "application/xml"},
			Produces: []string{"application/json"},
		}

		result := RouteToSpecOperation(route)

		if len(result.Consumes) != 2 {
			t.Errorf("Expected 2 consumes, got %d", len(result.Consumes))
		}
		if len(result.Produces) != 1 {
			t.Errorf("Expected 1 produces, got %d", len(result.Produces))
		}
	})

	t.Run("converts deprecated flag", func(t *testing.T) {
		route := &domain.Route{
			Deprecated: true,
		}

		result := RouteToSpecOperation(route)

		if !result.Deprecated {
			t.Error("Expected operation to be marked as deprecated")
		}
	})
}

func TestParameterToSpec(t *testing.T) {
	t.Run("converts simple parameter", func(t *testing.T) {
		param := domain.Parameter{
			Name:        "id",
			In:          "path",
			Type:        "integer",
			Required:    true,
			Description: "User ID",
			Format:      "int64",
		}

		result := ParameterToSpec(param)

		if result.Name != "id" {
			t.Errorf("Expected name 'id', got %s", result.Name)
		}
		if result.In != "path" {
			t.Errorf("Expected in 'path', got %s", result.In)
		}
		if result.Type != "integer" {
			t.Errorf("Expected type 'integer', got %s", result.Type)
		}
		if result.Format != "int64" {
			t.Errorf("Expected format 'int64', got %s", result.Format)
		}
		if !result.Required {
			t.Error("Expected parameter to be required")
		}
	})

	t.Run("converts array parameter", func(t *testing.T) {
		param := domain.Parameter{
			Name:     "ids",
			In:       "query",
			Type:     "array",
			Required: false,
			Items: &domain.Items{
				Type:   "integer",
				Format: "int64",
			},
		}

		result := ParameterToSpec(param)

		if result.Type != "array" {
			t.Errorf("Expected type 'array', got %s", result.Type)
		}
		if result.Items == nil {
			t.Fatal("Expected items to be set")
		}
		if result.Items.Type != "integer" {
			t.Errorf("Expected items type 'integer', got %s", result.Items.Type)
		}
	})

	t.Run("converts parameter with schema", func(t *testing.T) {
		param := domain.Parameter{
			Name:     "body",
			In:       "body",
			Required: true,
			Schema: &domain.Schema{
				Type: "object",
				Ref:  "#/definitions/User",
			},
		}

		result := ParameterToSpec(param)

		if result.Schema == nil {
			t.Fatal("Expected schema to be set")
		}
		if len(result.Schema.Type) == 0 || result.Schema.Type[0] != "object" {
			t.Errorf("Expected schema type 'object', got %v", result.Schema.Type)
		}
	})

	t.Run("converts parameter with default value", func(t *testing.T) {
		param := domain.Parameter{
			Name:    "limit",
			In:      "query",
			Type:    "integer",
			Default: 10,
		}

		result := ParameterToSpec(param)

		if result.Default != 10 {
			t.Errorf("Expected default 10, got %v", result.Default)
		}
	})

	t.Run("converts parameter with enum", func(t *testing.T) {
		param := domain.Parameter{
			Name: "status",
			In:   "query",
			Type: "string",
			Enum: []interface{}{"active", "inactive", "pending"},
		}

		result := ParameterToSpec(param)

		if len(result.Enum) != 3 {
			t.Errorf("Expected 3 enum values, got %d", len(result.Enum))
		}
	})
}

func TestResponseToSpec(t *testing.T) {
	t.Run("converts simple response", func(t *testing.T) {
		resp := domain.Response{
			Description: "Success",
		}

		result := ResponseToSpec(resp)

		if result.Description != "Success" {
			t.Errorf("Expected description 'Success', got %s", result.Description)
		}
	})

	t.Run("converts response with schema", func(t *testing.T) {
		resp := domain.Response{
			Description: "Success",
			Schema: &domain.Schema{
				Type: "object",
				Ref:  "#/definitions/User",
			},
		}

		result := ResponseToSpec(resp)

		if result.Schema == nil {
			t.Fatal("Expected schema to be set")
		}
		if len(result.Schema.Type) == 0 || result.Schema.Type[0] != "object" {
			t.Errorf("Expected schema type 'object', got %v", result.Schema.Type)
		}
	})

	t.Run("converts response with headers", func(t *testing.T) {
		resp := domain.Response{
			Description: "Success",
			Headers: map[string]domain.Header{
				"X-Request-Id": {
					Type:        "string",
					Description: "Request ID",
				},
				"X-Rate-Limit": {
					Type:        "integer",
					Format:      "int32",
					Description: "Rate limit",
				},
			},
		}

		result := ResponseToSpec(resp)

		if len(result.Headers) != 2 {
			t.Fatalf("Expected 2 headers, got %d", len(result.Headers))
		}

		if header, ok := result.Headers["X-Request-Id"]; !ok {
			t.Error("Expected X-Request-Id header")
		} else if header.Type != "string" {
			t.Errorf("Expected X-Request-Id type 'string', got %s", header.Type)
		}
	})
}

func TestSchemaToSpec(t *testing.T) {
	t.Run("nil schema returns nil", func(t *testing.T) {
		result := SchemaToSpec(nil)
		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})

	t.Run("converts simple schema", func(t *testing.T) {
		schema := &domain.Schema{
			Type:        "string",
			Description: "A string value",
		}

		result := SchemaToSpec(schema)

		if len(result.Type) == 0 || result.Type[0] != "string" {
			t.Errorf("Expected type 'string', got %v", result.Type)
		}
		if result.Description != "A string value" {
			t.Errorf("Expected description 'A string value', got %s", result.Description)
		}
	})

	t.Run("converts schema with reference", func(t *testing.T) {
		schema := &domain.Schema{
			Ref: "#/definitions/User",
		}

		result := SchemaToSpec(schema)

		if result.Ref.String() != "#/definitions/User" {
			t.Errorf("Expected ref '#/definitions/User', got %s", result.Ref.String())
		}
	})

	t.Run("converts array schema", func(t *testing.T) {
		schema := &domain.Schema{
			Type: "array",
			Items: &domain.Schema{
				Type: "string",
			},
		}

		result := SchemaToSpec(schema)

		if len(result.Type) == 0 || result.Type[0] != "array" {
			t.Errorf("Expected type 'array', got %v", result.Type)
		}
		if result.Items == nil {
			t.Fatal("Expected items to be set")
		}
		if result.Items.Schema == nil {
			t.Fatal("Expected items schema to be set")
		}
		if len(result.Items.Schema.Type) == 0 || result.Items.Schema.Type[0] != "string" {
			t.Errorf("Expected items type 'string', got %v", result.Items.Schema.Type)
		}
	})

	t.Run("converts object schema with properties", func(t *testing.T) {
		schema := &domain.Schema{
			Type: "object",
			Properties: map[string]*domain.Schema{
				"name": {
					Type:        "string",
					Description: "User name",
				},
				"age": {
					Type:        "integer",
					Description: "User age",
				},
			},
			Required: []string{"name"},
		}

		result := SchemaToSpec(schema)

		if len(result.Type) == 0 || result.Type[0] != "object" {
			t.Errorf("Expected type 'object', got %v", result.Type)
		}
		if len(result.Properties) != 2 {
			t.Fatalf("Expected 2 properties, got %d", len(result.Properties))
		}
		if _, ok := result.Properties["name"]; !ok {
			t.Error("Expected 'name' property")
		}
		if _, ok := result.Properties["age"]; !ok {
			t.Error("Expected 'age' property")
		}
		if len(result.Required) != 1 || result.Required[0] != "name" {
			t.Errorf("Expected required ['name'], got %v", result.Required)
		}
	})
}

func TestHeaderToSpec(t *testing.T) {
	t.Run("converts header", func(t *testing.T) {
		header := domain.Header{
			Type:        "string",
			Description: "Request ID",
			Format:      "uuid",
		}

		result := HeaderToSpec(header)

		if result.Type != "string" {
			t.Errorf("Expected type 'string', got %s", result.Type)
		}
		if result.Description != "Request ID" {
			t.Errorf("Expected description 'Request ID', got %s", result.Description)
		}
		if result.Format != "uuid" {
			t.Errorf("Expected format 'uuid', got %s", result.Format)
		}
	})
}
