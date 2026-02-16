package domain

import (
	"go/ast"
	"regexp"
	"strings"

	"github.com/go-openapi/spec"
)

const (
	// ARRAY represent a array value.
	ARRAY = "array"
	// OBJECT represent a object value.
	OBJECT = "object"
	// PRIMITIVE represent a primitive value.
	PRIMITIVE = "primitive"
	// BOOLEAN represent a boolean value.
	BOOLEAN = "boolean"
	// INTEGER represent a integer value.
	INTEGER = "integer"
	// NUMBER represent a number value.
	NUMBER = "number"
	// STRING represent a string value.
	STRING = "string"
	// FUNC represent a function value.
	FUNC = "func"
	// ERROR represent a error value.
	ERROR = "error"
	// INTERFACE represent a interface value.
	INTERFACE = "interface{}"
	// ANY represent a any value.
	ANY = "any"
	// NIL represent a empty value.
	NIL = "nil"
)

const (
	// IgnoreNameOverridePrefix character used in name comment to override type name
	IgnoreNameOverridePrefix = '!'
)

var overrideNameRegex = regexp.MustCompile(`(?i)^@name\s+(\S+)`)

// IsGolangPrimitiveType checks if a type is a Go primitive type.
// This only checks for basic Go types. For extended primitives (time.Time, UUID, decimal),
// use IsExtendedPrimitiveType instead.
func IsGolangPrimitiveType(typeName string) bool {
	switch typeName {
	case "uint",
		"int",
		"uint8",
		"int8",
		"uint16",
		"int16",
		"byte",
		"uint32",
		"int32",
		"rune",
		"uint64",
		"int64",
		"float32",
		"float64",
		"bool",
		"string":
		return true
	}

	return false
}

// IsExtendedPrimitiveType checks if a type should be treated as a primitive in OpenAPI,
// including time.Time, UUID, and decimal types that are commonly treated as primitives.
func IsExtendedPrimitiveType(typeName string) bool {
	// Strip pointer prefix for checking
	cleanType := strings.TrimPrefix(typeName, "*")

	// Check basic Go primitives first
	if IsGolangPrimitiveType(cleanType) {
		return true
	}

	// Check extended primitives (commonly treated as primitives in OpenAPI)
	switch cleanType {
	case "time.Time",
		"decimal.Decimal",
		"github.com/shopspring/decimal.Decimal",
		"types.UUID",
		"uuid.UUID",
		"github.com/griffnb/core/lib/types.UUID",
		"github.com/google/uuid.UUID":
		return true
	}

	return false
}

func ignoreNameOverride(name string) bool {
	return len(name) != 0 && name[0] == IgnoreNameOverridePrefix
}

func nameOverride(commentGroup *ast.CommentGroup) string {
	if commentGroup == nil {
		return ""
	}

	// get alias from comment '// @name '
	for _, comment := range commentGroup.List {
		trimmedComment := strings.TrimSpace(strings.TrimLeft(comment.Text, "/"))
		texts := overrideNameRegex.FindStringSubmatch(trimmedComment)
		if len(texts) > 1 {
			return texts[1]
		}
	}

	return ""
}

func fullTypeName(parts ...string) string {
	return strings.Join(parts, ".")
}

// TransToValidPrimitiveSchema transfer golang basic type to swagger schema with format considered.
// Handles both basic Go types and extended primitives (time.Time, UUID, decimal).
func TransToValidPrimitiveSchema(typeName string) *spec.Schema {
	// Strip pointer prefix for processing
	cleanType := strings.TrimPrefix(typeName, "*")

	switch cleanType {
	case "int", "uint":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{INTEGER}}}
	case "uint8", "int8", "uint16", "int16", "byte", "int32", "uint32", "rune":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{INTEGER}, Format: "int32"}}
	case "uint64", "int64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{INTEGER}, Format: "int64"}}
	case "float32":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{NUMBER}, Format: "float"}}
	case "float64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{NUMBER}, Format: "double"}}
	case "bool":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{BOOLEAN}}}
	case "string":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{STRING}}}
	// Extended primitives
	case "time.Time":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{STRING}, Format: "date-time"}}
	case "types.UUID", "uuid.UUID", "github.com/griffnb/core/lib/types.UUID", "github.com/google/uuid.UUID":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{STRING}, Format: "uuid"}}
	case "decimal.Decimal", "github.com/shopspring/decimal.Decimal":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{NUMBER}}}
	}
	return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{typeName}}}
}
