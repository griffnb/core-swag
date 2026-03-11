// Package schemautil provides shared schema utility functions without circular dependencies.
package schemautil

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

// IsPrimitiveType determines whether the type name is a primitive type.
func IsPrimitiveType(typeName string) bool {
	switch typeName {
	case STRING, NUMBER, INTEGER, BOOLEAN, ARRAY, OBJECT, FUNC:
		return true
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
