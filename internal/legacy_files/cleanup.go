package swag

import (
	"github.com/go-openapi/spec"
	"github.com/swaggo/swag/internal/schema"
)

// RemoveUnusedDefinitions removes schema definitions that are not referenced anywhere in the Swagger spec.
// Deprecated: This function is now an alias to schema.RemoveUnusedDefinitions for backward compatibility.
func RemoveUnusedDefinitions(swagger *spec.Swagger) {
	schema.RemoveUnusedDefinitions(swagger)
}

// getRefName extracts the definition name from a $ref string like "#/definitions/ModelName"
// Kept for backward compatibility with existing tests.
func getRefName(ref string) string {
	const prefix = "#/definitions/"
	if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ""
}
