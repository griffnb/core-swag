package route

import (
	"testing"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/parser/route/domain"
)

func TestRegisterRoutes(t *testing.T) {
	t.Run("registers single route", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
			},
		}

		routes := []*domain.Route{
			{
				Method:      "GET",
				Path:        "/users",
				Summary:     "List users",
				Description: "Returns all users",
			},
		}

		err := service.RegisterRoutes(swagger, routes, false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(swagger.Paths.Paths) != 1 {
			t.Fatalf("Expected 1 path, got %d", len(swagger.Paths.Paths))
		}

		pathItem, ok := swagger.Paths.Paths["/users"]
		if !ok {
			t.Fatal("Expected /users path to exist")
		}

		if pathItem.Get == nil {
			t.Fatal("Expected GET operation to exist")
		}

		if pathItem.Get.Summary != "List users" {
			t.Errorf("Expected summary 'List users', got %s", pathItem.Get.Summary)
		}
	})

	t.Run("registers multiple routes to same path", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
			},
		}

		routes := []*domain.Route{
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users",
			},
			{
				Method:  "POST",
				Path:    "/users",
				Summary: "Create user",
			},
		}

		err := service.RegisterRoutes(swagger, routes, false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		pathItem := swagger.Paths.Paths["/users"]

		if pathItem.Get == nil {
			t.Error("Expected GET operation")
		}
		if pathItem.Post == nil {
			t.Error("Expected POST operation")
		}
	})

	t.Run("registers routes to different paths", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
			},
		}

		routes := []*domain.Route{
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users",
			},
			{
				Method:  "GET",
				Path:    "/posts",
				Summary: "List posts",
			},
		}

		err := service.RegisterRoutes(swagger, routes, false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(swagger.Paths.Paths) != 2 {
			t.Fatalf("Expected 2 paths, got %d", len(swagger.Paths.Paths))
		}

		if _, ok := swagger.Paths.Paths["/users"]; !ok {
			t.Error("Expected /users path")
		}
		if _, ok := swagger.Paths.Paths["/posts"]; !ok {
			t.Error("Expected /posts path")
		}
	})

	t.Run("handles duplicate routes in non-strict mode", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
			},
		}

		routes := []*domain.Route{
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users v1",
			},
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users v2",
			},
		}

		// Should not error in non-strict mode, just warn
		err := service.RegisterRoutes(swagger, routes, false)
		if err != nil {
			t.Fatalf("Expected no error in non-strict mode, got %v", err)
		}

		// Last one wins
		pathItem := swagger.Paths.Paths["/users"]
		if pathItem.Get.Summary != "List users v2" {
			t.Errorf("Expected last route to win, got %s", pathItem.Get.Summary)
		}
	})

	t.Run("errors on duplicate routes in strict mode", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
			},
		}

		routes := []*domain.Route{
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users v1",
			},
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users v2",
			},
		}

		// Should error in strict mode
		err := service.RegisterRoutes(swagger, routes, true)
		if err == nil {
			t.Fatal("Expected error in strict mode for duplicate routes")
		}
	})

	t.Run("initializes swagger.Paths if nil", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{}

		routes := []*domain.Route{
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users",
			},
		}

		err := service.RegisterRoutes(swagger, routes, false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if swagger.Paths == nil {
			t.Fatal("Expected swagger.Paths to be initialized")
		}

		if len(swagger.Paths.Paths) != 1 {
			t.Fatalf("Expected 1 path, got %d", len(swagger.Paths.Paths))
		}
	})
}

func TestRegisterRoute(t *testing.T) {
	t.Run("registers route with extensions", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
			},
		}

		route := &domain.Route{
			Method:       "GET",
			Path:         "/users/{id}",
			Summary:      "Get user",
			FilePath:     "/src/handlers/user.go",
			FunctionName: "GetUser",
			LineNumber:   42,
		}

		err := service.registerRoute(swagger, route, false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		pathItem := swagger.Paths.Paths["/users/{id}"]
		if pathItem.Get == nil {
			t.Fatal("Expected GET operation")
		}

		if pathItem.Get.Extensions["x-path"] != "/src/handlers/user.go" {
			t.Errorf("Expected x-path extension, got %v", pathItem.Get.Extensions["x-path"])
		}
		if pathItem.Get.Extensions["x-function"] != "GetUser" {
			t.Errorf("Expected x-function extension, got %v", pathItem.Get.Extensions["x-function"])
		}
		if pathItem.Get.Extensions["x-line"] != 42 {
			t.Errorf("Expected x-line extension, got %v", pathItem.Get.Extensions["x-line"])
		}
	})

	t.Run("filters invalid path parameters", func(t *testing.T) {
		service := NewService(nil, nil)
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
			},
		}

		route := &domain.Route{
			Method: "GET",
			Path:   "/users/{id}",
			Parameters: []domain.Parameter{
				{
					Name:     "id",
					In:       "path",
					Type:     "integer",
					Required: true,
				},
				{
					Name:     "otherId",
					In:       "path",
					Type:     "integer",
					Required: true,
				},
				{
					Name:     "name",
					In:       "query",
					Type:     "string",
					Required: false,
				},
			},
		}

		err := service.registerRoute(swagger, route, false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		pathItem := swagger.Paths.Paths["/users/{id}"]
		if pathItem.Get == nil {
			t.Fatal("Expected GET operation")
		}

		// Should have 2 parameters: id (path) and name (query)
		// otherId (path) should be filtered out
		if len(pathItem.Get.Parameters) != 2 {
			t.Fatalf("Expected 2 parameters, got %d", len(pathItem.Get.Parameters))
		}

		// Check that we have id and name, but not otherId
		hasId := false
		hasName := false
		hasOtherId := false

		for _, param := range pathItem.Get.Parameters {
			if param.Name == "id" {
				hasId = true
			}
			if param.Name == "name" {
				hasName = true
			}
			if param.Name == "otherId" {
				hasOtherId = true
			}
		}

		if !hasId {
			t.Error("Expected 'id' parameter")
		}
		if !hasName {
			t.Error("Expected 'name' parameter")
		}
		if hasOtherId {
			t.Error("Expected 'otherId' parameter to be filtered out")
		}
	})
}

func TestRefRouteMethodOp(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   bool // true if should return non-nil
	}{
		{"GET", "GET", true},
		{"POST", "POST", true},
		{"PUT", "PUT", true},
		{"DELETE", "DELETE", true},
		{"PATCH", "PATCH", true},
		{"HEAD", "HEAD", true},
		{"OPTIONS", "OPTIONS", true},
		{"Invalid", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathItem := spec.PathItem{}
			result := refRouteMethodOp(&pathItem, tt.method)

			if tt.want && result == nil {
				t.Errorf("Expected non-nil result for method %s", tt.method)
			}
			if !tt.want && result != nil {
				t.Errorf("Expected nil result for method %s", tt.method)
			}
		})
	}
}

func TestFilterValidPathParameters(t *testing.T) {
	t.Run("keeps all parameters for path without path params", func(t *testing.T) {
		params := []spec.Parameter{
			{ParamProps: spec.ParamProps{Name: "name", In: "query"}},
			{ParamProps: spec.ParamProps{Name: "age", In: "query"}},
		}

		result := filterValidPathParameters(params, "/users")

		if len(result) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(result))
		}
	})

	t.Run("keeps valid path parameters", func(t *testing.T) {
		params := []spec.Parameter{
			{ParamProps: spec.ParamProps{Name: "id", In: "path"}},
			{ParamProps: spec.ParamProps{Name: "name", In: "query"}},
		}

		result := filterValidPathParameters(params, "/users/{id}")

		if len(result) != 2 {
			t.Fatalf("Expected 2 parameters, got %d", len(result))
		}
	})

	t.Run("filters invalid path parameters", func(t *testing.T) {
		params := []spec.Parameter{
			{ParamProps: spec.ParamProps{Name: "id", In: "path"}},
			{ParamProps: spec.ParamProps{Name: "otherId", In: "path"}},
			{ParamProps: spec.ParamProps{Name: "name", In: "query"}},
		}

		result := filterValidPathParameters(params, "/users/{id}")

		if len(result) != 2 {
			t.Fatalf("Expected 2 parameters (id and name), got %d", len(result))
		}

		// Check that otherId was filtered out
		for _, param := range result {
			if param.Name == "otherId" {
				t.Error("Expected otherId to be filtered out")
			}
		}
	})

	t.Run("keeps multiple valid path parameters", func(t *testing.T) {
		params := []spec.Parameter{
			{ParamProps: spec.ParamProps{Name: "userId", In: "path"}},
			{ParamProps: spec.ParamProps{Name: "postId", In: "path"}},
			{ParamProps: spec.ParamProps{Name: "name", In: "query"}},
		}

		result := filterValidPathParameters(params, "/users/{userId}/posts/{postId}")

		if len(result) != 3 {
			t.Fatalf("Expected 3 parameters, got %d", len(result))
		}
	})
}
