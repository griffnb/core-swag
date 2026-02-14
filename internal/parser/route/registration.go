package route

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/swaggo/swag/console"
	"github.com/swaggo/swag/internal/parser/route/domain"
)

// RegisterRoutes registers routes to swagger.Paths
// strict controls whether to error on duplicate routes
func (s *Service) RegisterRoutes(swagger *spec.Swagger, routes []*domain.Route, strict bool) error {
	if swagger.Paths == nil {
		swagger.Paths = &spec.Paths{
			Paths: make(map[string]spec.PathItem),
		}
	}

	for _, route := range routes {
		if err := s.registerRoute(swagger, route, strict); err != nil {
			return err
		}
	}

	return nil
}

// registerRoute registers a single route to swagger.Paths
func (s *Service) registerRoute(swagger *spec.Swagger, route *domain.Route, strict bool) error {
	// Get or create path item
	pathItem, exists := swagger.Paths.Paths[route.Path]
	if !exists {
		pathItem = spec.PathItem{}
	}

	// Get reference to the operation for this HTTP method
	op := refRouteMethodOp(&pathItem, route.Method)
	if op == nil {
		return fmt.Errorf("invalid HTTP method: %s", route.Method)
	}

	// Check if operation already exists
	if *op != nil {
		err := fmt.Errorf("route %s %s is declared multiple times", route.Method, route.Path)
		if strict {
			return err
		}
		console.Logger.Debug("warning: %s\n", err)
	}

	// Convert domain.Route to spec.Operation
	specOp := RouteToSpecOperation(route)
	if specOp == nil {
		return fmt.Errorf("failed to convert route to operation: %s %s", route.Method, route.Path)
	}

	// Filter parameters for path parameters only in this route
	// (if we had multiple routes with shared parameters, we need to filter)
	validParams := filterValidPathParameters(specOp.Parameters, route.Path)
	specOp.Parameters = validParams

	// Set the operation
	*op = specOp

	// Save the path item
	swagger.Paths.Paths[route.Path] = pathItem

	return nil
}

// refRouteMethodOp returns a pointer to the operation field for the given HTTP method
func refRouteMethodOp(item *spec.PathItem, method string) **spec.Operation {
	switch method {
	case http.MethodGet:
		return &item.Get
	case http.MethodPost:
		return &item.Post
	case http.MethodDelete:
		return &item.Delete
	case http.MethodPut:
		return &item.Put
	case http.MethodPatch:
		return &item.Patch
	case http.MethodHead:
		return &item.Head
	case http.MethodOptions:
		return &item.Options
	default:
		return nil
	}
}

// filterValidPathParameters filters out path parameters that don't exist in the path
func filterValidPathParameters(params []spec.Parameter, path string) []spec.Parameter {
	var validParams []spec.Parameter

	for _, param := range params {
		// Only filter path parameters
		if param.In == "path" {
			// Check if the parameter name is actually in the path
			if !strings.Contains(path, "{"+param.Name+"}") {
				// Path parameter not in this specific path, skip it
				continue
			}
		}

		// Keep all non-path parameters and valid path parameters
		validParams = append(validParams, param)
	}

	return validParams
}
