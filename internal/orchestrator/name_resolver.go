// Package orchestrator coordinates all services to generate OpenAPI documentation.
package orchestrator

import (
	"strings"

	"github.com/griffnb/core-swag/internal/model"
	"github.com/griffnb/core-swag/internal/registry"
)

// registryNameResolver implements model.DefinitionNameResolver using the
// registry to determine whether a type is unique (short name) or NotUnique
// (full-path name). This bridges the model layer to the registry without
// creating a direct dependency.
type registryNameResolver struct {
	registry *registry.Service
}

// newRegistryNameResolver creates a resolver backed by the given registry.
func newRegistryNameResolver(reg *registry.Service) model.DefinitionNameResolver {
	return &registryNameResolver{registry: reg}
}

// ResolveDefinitionName returns the canonical definition name for the type
// identified by fullTypePath (e.g., "github.com/user/project/internal/constants.Role").
//
// Strategy:
//  1. Extract short name (e.g., "constants.Role") and look it up in the registry.
//     If found, TypeName() returns short for unique types, full-path for NotUnique.
//  2. If not found by short name, compute the full-path key and try again.
//  3. Fallback to the short name.
func (r *registryNameResolver) ResolveDefinitionName(fullTypePath string) string {
	// Extract short name: "github.com/user/project/internal/constants.Role" → "constants.Role"
	shortName := extractShortTypeName(fullTypePath)

	// Try short-name lookup (works for unique types whose registry key is the short name)
	typeDef := r.registry.FindTypeSpecByName(shortName)
	if typeDef != nil {
		return typeDef.TypeName()
	}

	// Try full-path key lookup (works for NotUnique types)
	fullPathKey := makeFullPathDefName2(fullTypePath)
	typeDef = r.registry.FindTypeSpecByName(fullPathKey)
	if typeDef != nil {
		return typeDef.TypeName()
	}

	// Not in registry — use short name
	return shortName
}

// extractShortTypeName extracts "package.TypeName" from a full module path.
// "github.com/user/project/internal/constants.Role" → "constants.Role"
func extractShortTypeName(fullPath string) string {
	if !strings.Contains(fullPath, "/") {
		return fullPath
	}
	lastSlash := strings.LastIndex(fullPath, "/")
	return fullPath[lastSlash+1:]
}

// makeFullPathDefName2 converts a full type path to the definition name format
// used by TypeSpecDef.TypeName() for NotUnique types. Same algorithm as
// makeFullPathDefName but takes a single combined string.
// "github.com/chargebee/chargebee-go/v3/enum.Source" →
// "github_com_chargebee_chargebee-go_v3_enum.Source"
func makeFullPathDefName2(fullTypePath string) string {
	lastDot := strings.LastIndex(fullTypePath, ".")
	if lastDot == -1 {
		return fullTypePath
	}
	pkgPath := fullTypePath[:lastDot]
	typeName := fullTypePath[lastDot+1:]

	sanitized := strings.Map(func(r rune) rune {
		if r == '\\' || r == '/' || r == '.' {
			return '_'
		}
		return r
	}, pkgPath)

	return sanitized + "." + typeName
}
