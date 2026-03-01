package orchestrator

import (
	"strings"

	routedomain "github.com/griffnb/core-swag/internal/parser/route/domain"
)

// CollectReferencedTypes walks all routes and returns the unique set of type names
// referenced in $ref strings from parameters, responses, and nested schemas.
// The returned map keys are type names with the "#/definitions/" prefix stripped.
func CollectReferencedTypes(routes []*routedomain.Route) map[string]bool {
	refs := make(map[string]bool)
	for _, route := range routes {
		if route == nil {
			continue
		}
		collectRefsFromRoute(route, refs)
	}
	return refs
}

// collectRefsFromRoute extracts all $ref type names from a single route.
func collectRefsFromRoute(r *routedomain.Route, refs map[string]bool) {
	for i := range r.Parameters {
		if r.Parameters[i].Schema != nil {
			collectRefsFromSchema(r.Parameters[i].Schema, refs)
		}
	}
	for _, resp := range r.Responses {
		if resp.Schema != nil {
			collectRefsFromSchema(resp.Schema, refs)
		}
	}
}

// collectRefsFromSchema recursively walks a schema tree and collects all $ref type names.
func collectRefsFromSchema(s *routedomain.Schema, refs map[string]bool) {
	if s == nil {
		return
	}
	if s.Ref != "" {
		typeName := strings.TrimPrefix(s.Ref, "#/definitions/")
		if typeName != "" {
			refs[typeName] = true
		}
	}
	if s.Items != nil {
		collectRefsFromSchema(s.Items, refs)
	}
	if s.AdditionalProperties != nil {
		collectRefsFromSchema(s.AdditionalProperties, refs)
	}
	for _, prop := range s.Properties {
		collectRefsFromSchema(prop, refs)
	}
	for _, allOf := range s.AllOf {
		collectRefsFromSchema(allOf, refs)
	}
}
