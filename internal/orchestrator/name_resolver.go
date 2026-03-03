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
//  1. Extract short name and look it up. If found and unique, return short name.
//  2. If found but NotUnique, verify the result matches the requested package path.
//     The short-name fallback in FindTypeSpecByName may return a different package's
//     type when multiple packages define the same type name (e.g., "enum.Source"
//     exists in both v3/enum and v3/models/recordedpurchase/enum).
//  3. If not found or wrong package, compute full-path key and try that.
//  4. Fallback: use full-path key if we know the type is NotUnique, otherwise short name.
func (r *registryNameResolver) ResolveDefinitionName(fullTypePath string) string {
	shortName := extractShortTypeName(fullTypePath)
	fullPathKey := makeFullPathDefName2(fullTypePath)

	typeDef := r.registry.FindTypeSpecByName(shortName)
	if typeDef != nil {
		result := typeDef.TypeName()
		// Unique type — short name is the canonical form
		if result == shortName {
			return result
		}
		// NotUnique — verify the returned type matches the requested package path.
		// If it does, use it. If not, fall through to exact full-path lookup.
		if result == fullPathKey {
			return result
		}
	}

	// Try exact full-path key lookup
	typeDef = r.registry.FindTypeSpecByName(fullPathKey)
	if typeDef != nil {
		return typeDef.TypeName()
	}

	// Not in registry by exact key — if the short-name search found a different
	// package's type, this type is also NotUnique, so use the full-path key.
	if typeDef := r.registry.FindTypeSpecByName(shortName); typeDef != nil {
		return fullPathKey
	}

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
