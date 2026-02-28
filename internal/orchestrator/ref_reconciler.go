package orchestrator

import "strings"

// makeFullPathDefName builds the full-path definition name from a package path and type name.
// This matches the algorithm in TypeSpecDef.TypeName() for NotUnique types:
// replace \, /, . in pkgPath with _, then join with type name using ".".
// Example: ("github.com/chargebee/chargebee-go/v3/enum", "Source") → "github_com_chargebee_chargebee-go_v3_enum.Source"
func makeFullPathDefName(pkgPath, typeName string) string {
	sanitized := strings.Map(func(r rune) rune {
		if r == '\\' || r == '/' || r == '.' {
			return '_'
		}
		return r
	}, pkgPath)
	return sanitized + "." + typeName
}
