package orchestrator

import (
	"fmt"
	"strings"

	routedomain "github.com/griffnb/core-swag/internal/parser/route/domain"
)

// CollectReferencedTypes walks all routes and returns the unique set of type names
// referenced in $ref strings from parameters, responses, and nested schemas.
// The returned map keys are type names with the "#/definitions/" prefix stripped.
// Values are source locations describing where the ref was first encountered
// (e.g., "POST /users (controllers.go:42)").
func CollectReferencedTypes(routes []*routedomain.Route) map[string]string {
	refs := make(map[string]string)
	for _, route := range routes {
		if route == nil {
			continue
		}
		source := routeSource(route)
		collectRefsFromRoute(route, refs, source)
	}
	return refs
}

// routeSource builds a human-readable location string for a route.
func routeSource(r *routedomain.Route) string {
	parts := []string{}
	if r.Method != "" {
		parts = append(parts, r.Method)
	}
	if r.Path != "" {
		parts = append(parts, r.Path)
	}
	loc := strings.Join(parts, " ")
	if r.FunctionName != "" {
		loc += " → " + r.FunctionName
	}
	if r.FilePath != "" {
		file := r.FilePath
		// Use just the filename, not full path
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			file = file[idx+1:]
		}
		if r.LineNumber > 0 {
			loc += fmt.Sprintf(" (%s:%d)", file, r.LineNumber)
		} else {
			loc += fmt.Sprintf(" (%s)", file)
		}
	}
	return loc
}

// collectRefsFromRoute extracts all $ref type names from a single route.
func collectRefsFromRoute(r *routedomain.Route, refs map[string]string, source string) {
	for i := range r.Parameters {
		if r.Parameters[i].Schema != nil {
			collectRefsFromSchema(r.Parameters[i].Schema, refs, source)
		}
	}
	for _, resp := range r.Responses {
		if resp.Schema != nil {
			collectRefsFromSchema(resp.Schema, refs, source)
		}
	}
}

// collectRefsFromSchema recursively walks a schema tree and collects all $ref type names.
func collectRefsFromSchema(s *routedomain.Schema, refs map[string]string, source string) {
	if s == nil {
		return
	}
	if s.Ref != "" {
		typeName := strings.TrimPrefix(s.Ref, "#/definitions/")
		if typeName != "" {
			if _, exists := refs[typeName]; !exists {
				refs[typeName] = source
			}
		}
	}
	if s.Items != nil {
		collectRefsFromSchema(s.Items, refs, source)
	}
	if s.AdditionalProperties != nil {
		collectRefsFromSchema(s.AdditionalProperties, refs, source)
	}
	for _, prop := range s.Properties {
		collectRefsFromSchema(prop, refs, source)
	}
	for _, allOf := range s.AllOf {
		collectRefsFromSchema(allOf, refs, source)
	}
}
