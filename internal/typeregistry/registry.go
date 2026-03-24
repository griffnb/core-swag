// Package typeregistry centralizes custom type-to-OpenAPI mappings.
// All custom external types (UUID, Decimal, Time, URN, etc.) are registered
// here so consumers don't need to maintain their own hardcoded maps.
package typeregistry

import (
	"strings"

	"github.com/go-openapi/spec"
)

// TypeEntry maps a custom Go type to its OpenAPI schema type and format.
type TypeEntry struct {
	SchemaType string // "string", "number", "integer", "boolean", "object"
	Format     string // "uuid", "date-time", "uri", "byte", ""
}

// registry is the central map of custom types to their OpenAPI representations.
// Go primitives (int, string, bool, etc.) are NOT included here — only extended
// types that need special OpenAPI mapping.
var registry = map[string]TypeEntry{
	// Time
	"time.Time": {SchemaType: "string", Format: "date-time"},

	// UUID variants
	"types.UUID":                             {SchemaType: "string", Format: "uuid"},
	"uuid.UUID":                              {SchemaType: "string", Format: "uuid"},
	"github.com/google/uuid.UUID":            {SchemaType: "string", Format: "uuid"},
	"github.com/griffnb/core/lib/types.UUID": {SchemaType: "string", Format: "uuid"},

	// URN variants
	"types.URN":                             {SchemaType: "string", Format: "uri"},
	"github.com/griffnb/core/lib/types.URN": {SchemaType: "string", Format: "uri"},

	// Decimal variants
	"decimal.Decimal":                       {SchemaType: "string", Format: ""},
	"github.com/shopspring/decimal.Decimal": {SchemaType: "string", Format: ""},

	// JSON
	"json.RawMessage":          {SchemaType: "object", Format: ""},
	"encoding/json.RawMessage": {SchemaType: "object", Format: ""},

	// Byte arrays
	"[]byte":  {SchemaType: "string", Format: "byte"},
	"[]uint8": {SchemaType: "string", Format: "byte"},
}

// Lookup returns the TypeEntry for a custom type. Strips leading `*` before matching.
// Returns false if the type is not a registered custom type.
func Lookup(typeName string) (TypeEntry, bool) {
	clean := strings.TrimPrefix(typeName, "*")
	entry, ok := registry[clean]
	return entry, ok
}

// IsExtendedPrimitive returns true if the type is a registered custom type
// that should be treated as a primitive in OpenAPI (not a model/$ref).
// Does NOT include basic Go primitives — only extended types like UUID, Time, Decimal.
func IsExtendedPrimitive(typeName string) bool {
	_, ok := Lookup(typeName)
	return ok
}

// ToSchema builds an OpenAPI spec.Schema for a registered custom type.
// Returns nil if the type is not registered.
func ToSchema(typeName string) *spec.Schema {
	entry, ok := Lookup(typeName)
	if !ok {
		return nil
	}
	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{entry.SchemaType},
		},
	}
	if entry.Format != "" {
		schema.Format = entry.Format
	}
	return schema
}
