package schema

import (
	"errors"

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
)

// IsSimplePrimitiveType determines whether the type name is a simple primitive type.
func IsSimplePrimitiveType(typeName string) bool {
	switch typeName {
	case STRING, NUMBER, INTEGER, BOOLEAN:
		return true
	}
	return false
}

// IsPrimitiveType determines whether the type name is a primitive type.
func IsPrimitiveType(typeName string) bool {
	switch typeName {
	case STRING, NUMBER, INTEGER, BOOLEAN, ARRAY, OBJECT, FUNC:
		return true
	}
	return false
}

// IsComplexSchema determines whether a schema is complex and should be a ref schema.
func IsComplexSchema(schema *spec.Schema) bool {
	// a enum type should be complex
	if len(schema.Enum) > 0 {
		return true
	}

	// a deep array type is complex, how to determine deep? here more than 2, for example: [][]object,[][][]int
	if len(schema.Type) > 2 {
		return true
	}

	// Object included, such as Object or []Object
	for _, st := range schema.Type {
		if st == OBJECT {
			return true
		}
	}
	return false
}

// PrimitiveSchema builds a primitive schema.
func PrimitiveSchema(refType string) *spec.Schema {
	return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{refType}}}
}

// BuildCustomSchema builds custom schema specified by tag swaggertype.
func BuildCustomSchema(types []string) (*spec.Schema, error) {
	if len(types) == 0 {
		return nil, nil
	}

	switch types[0] {
	case PRIMITIVE:
		if len(types) == 1 {
			return nil, errors.New("need primitive type after primitive")
		}
		return BuildCustomSchema(types[1:])
	case ARRAY:
		if len(types) == 1 {
			return nil, errors.New("need array item type after array")
		}

		schema, err := BuildCustomSchema(types[1:])
		if err != nil {
			return nil, err
		}

		return spec.ArrayProperty(schema), nil
	case OBJECT:
		if len(types) == 1 {
			return PrimitiveSchema(types[0]), nil
		}

		schema, err := BuildCustomSchema(types[1:])
		if err != nil {
			return nil, err
		}

		return spec.MapProperty(schema), nil
	default:
		if !IsPrimitiveType(types[0]) {
			return nil, errors.New(types[0] + " is not basic types")
		}
		return PrimitiveSchema(types[0]), nil
	}
}

// MergeSchema merges schemas.
func MergeSchema(dst *spec.Schema, src *spec.Schema) *spec.Schema {
	if len(src.Type) > 0 {
		dst.Type = src.Type
	}
	if len(src.Properties) > 0 {
		dst.Properties = src.Properties
	}
	if src.Items != nil {
		dst.Items = src.Items
	}
	if src.AdditionalProperties != nil {
		dst.AdditionalProperties = src.AdditionalProperties
	}
	if len(src.Description) > 0 {
		dst.Description = src.Description
	}
	if src.Nullable {
		dst.Nullable = src.Nullable
	}
	if len(src.Format) > 0 {
		dst.Format = src.Format
	}
	if src.Default != nil {
		dst.Default = src.Default
	}
	if src.Example != nil {
		dst.Example = src.Example
	}
	if len(src.Extensions) > 0 {
		dst.Extensions = src.Extensions
	}
	if src.Maximum != nil {
		dst.Maximum = src.Maximum
	}
	if src.Minimum != nil {
		dst.Minimum = src.Minimum
	}
	if src.ExclusiveMaximum {
		dst.ExclusiveMaximum = src.ExclusiveMaximum
	}
	if src.ExclusiveMinimum {
		dst.ExclusiveMinimum = src.ExclusiveMinimum
	}
	if src.MaxLength != nil {
		dst.MaxLength = src.MaxLength
	}
	if src.MinLength != nil {
		dst.MinLength = src.MinLength
	}
	if len(src.Pattern) > 0 {
		dst.Pattern = src.Pattern
	}
	if src.MaxItems != nil {
		dst.MaxItems = src.MaxItems
	}
	if src.MinItems != nil {
		dst.MinItems = src.MinItems
	}
	if src.UniqueItems {
		dst.UniqueItems = src.UniqueItems
	}
	if src.MultipleOf != nil {
		dst.MultipleOf = src.MultipleOf
	}
	if len(src.Enum) > 0 {
		dst.Enum = src.Enum
	}
	if len(src.ExtraProps) > 0 {
		dst.ExtraProps = src.ExtraProps
	}
	return dst
}
