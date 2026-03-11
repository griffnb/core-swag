package model

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/console"
	"github.com/griffnb/core-swag/internal/schemautil"
)

type StructField struct {
	Name       string         `json:"name"`
	Type       types.Type     `json:"type"`
	TypeString string         `json:"type_string"` // For easier JSON serialization
	Tag        string         `json:"tag"`
	Fields     []*StructField `json:"fields"` // For nested structs
}

func (this *StructField) IsPublic() bool {
	_, ok := this.GetTags()["public"]
	return ok
}

func (this *StructField) GetTags() map[string]string {
	tags := strings.Split(this.Tag, " ")
	result := make(map[string]string)
	for _, tag := range tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			key := strings.Trim(parts[0], "`")
			value := strings.Trim(parts[1], "`")
			result[key] = strings.Trim(value, "\"")
		}
	}
	return result
}

// EffectiveTypeString returns the resolved type string for this field.
// Uses TypeString if set, falls back to Type.String() if Type is available.
func (this *StructField) EffectiveTypeString() string {
	if this.TypeString != "" {
		return this.TypeString
	}
	if this.Type != nil {
		return this.Type.String()
	}
	return ""
}

// NormalizedType returns the type string in short form (package.Type instead of
// full/module/path/package.Type). Handles pointers.
func (this *StructField) NormalizedType() string {
	return normalizeTypeName(this.EffectiveTypeString())
}

// ConstantFieldEnumType extracts the enum type parameter from IntConstantField[T]
// or StringConstantField[T]. Returns empty string if not a constant field.
func (this *StructField) ConstantFieldEnumType() string {
	return extractConstantFieldEnumType(this.EffectiveTypeString())
}

// IsGeneric returns true if this field is a generic wrapper type like
// StructField[T], IntConstantField[T], StringField[T], or any Field[T].
func (this *StructField) IsGeneric() bool {
	typeStr := this.EffectiveTypeString()
	return strings.Contains(typeStr, "StructField[") ||
		strings.Contains(typeStr, "IntConstantField[") ||
		strings.Contains(typeStr, "StringField[") ||
		strings.Contains(typeStr, "Field[")
}

// GenericTypeArg extracts the type parameter T from a generic wrapper like Field[T].
// Handles nested brackets like Field[map[string][]User].
// Returns error if the type string has no brackets or mismatched brackets.
func (this *StructField) GenericTypeArg() (string, error) {
	typeStr := this.EffectiveTypeString()
	return extractGenericTypeParameter(typeStr)
}

// IsPrimitive returns true if this field's type is a Go primitive or an extended
// primitive (time.Time, UUID, decimal.Decimal).
func (this *StructField) IsPrimitive() bool {
	primitives := map[string]bool{
		"string": true, "bool": true,
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"byte": true, "rune": true,
		"float32": true, "float64": true,
		"time.Time": true, "*time.Time": true,
		"decimal.Decimal": true, "*decimal.Decimal": true,
		"github.com/shopspring/decimal.Decimal": true, "*github.com/shopspring/decimal.Decimal": true,
		// UUID types
		"types.UUID": true, "*types.UUID": true,
		"uuid.UUID": true, "*uuid.UUID": true,
		"github.com/griffnb/core/lib/types.UUID": true, "*github.com/griffnb/core/lib/types.UUID": true,
		"github.com/google/uuid.UUID": true, "*github.com/google/uuid.UUID": true,
		// json.RawMessage is []byte representing raw JSON — treat as primitive (object)
		"json.RawMessage": true, "encoding/json.RawMessage": true,
		// []byte is a primitive represented as base64 string in OpenAPI
		"[]byte": true, "[]uint8": true,
	}
	return primitives[this.EffectiveTypeString()]
}

// IsAny returns true if this field's type is any or interface{}.
func (this *StructField) IsAny() bool {
	typeStr := this.EffectiveTypeString()
	if typeStr == "" {
		return false
	}
	if typeStr == "any" {
		return true
	}
	return strings.ReplaceAll(typeStr, " ", "") == "interface{}"
}

// IsFieldsWrapper returns true if this field is a fields package wrapper type
// like fields.StringField, fields.IntField, etc.
func (this *StructField) IsFieldsWrapper() bool {
	return strings.Contains(this.EffectiveTypeString(), "fields.")
}

// IsSwaggerPrimitive returns true if the field's Go type is a struct that should
// be treated as a primitive in Swagger (e.g., time.Time, decimal.Decimal, UUID).
func (this *StructField) IsSwaggerPrimitive() bool {
	if this.Type == nil {
		return false
	}
	t := this.Type
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	if named.Obj().Pkg() == nil {
		return false
	}

	pkgPath := named.Obj().Pkg().Path()
	typeName := named.Obj().Name()

	// Types that are structs in Go but should be primitives in Swagger
	primitiveTypes := map[string][]string{
		"time":                          {"Time"},
		"github.com/shopspring/decimal": {"Decimal"},
		"gopkg.in/guregu/null.v4":       {"String", "Int", "Float", "Bool", "Time"},
		"database/sql":                  {"NullString", "NullInt64", "NullFloat64", "NullBool", "NullTime"},
		"encoding/json":                 {"RawMessage"},
	}

	globalNames := []string{"UUID"}

	if typeNames, ok := primitiveTypes[pkgPath]; ok {
		for _, name := range typeNames {
			if typeName == name {
				return true
			}
		}
	} else {
		for _, globalName := range globalNames {
			if typeName == globalName {
				return true
			}
		}
	}

	return false
}

// IsGenericTypeArgStruct checks whether the first type argument of this field's
// generic type is a struct. Returns true if the type has no type arguments
// (not generic), or if the argument's underlying Go type is a struct.
func (this *StructField) IsGenericTypeArgStruct() bool {
	return isGenericTypeArgStruct(this.Type)
}

// IsUnderlyingStruct checks whether this field's underlying Go type is a struct.
// Unwraps pointers and named types. Returns true for unknown types (safe default).
func (this *StructField) IsUnderlyingStruct() bool {
	return isUnderlyingStruct(this.Type)
}

// PrimitiveSchema returns the OpenAPI schema for this field's primitive type.
func (this *StructField) PrimitiveSchema() *spec.Schema {
	return primitiveTypeToSchema(this.EffectiveTypeString())
}

// FieldsWrapperSchema returns the OpenAPI schema for a fields package wrapper type
// (StringField, IntField, IntConstantField[T], etc.).
func (this *StructField) FieldsWrapperSchema(enumLookup TypeEnumLookup) (*spec.Schema, []string, error) {
	return getPrimitiveSchemaForFieldType(this.EffectiveTypeString(), this.TypeString, enumLookup)
}

// BuildSchema builds an OpenAPI schema for this field's type.
// Checks for swaggertype tag first to allow user-specified type overrides.
// Applies struct tags (enums, format, constraints, etc.) to enrich the schema.
// For recursive types (arrays, maps), creates child StructField instances.
// Returns schema, list of nested struct type names for definition generation, and error.
func (this *StructField) BuildSchema(
	public bool,
	forceRequired bool,
	enumLookup TypeEnumLookup,
) (*spec.Schema, []string, error) {
	var nestedTypes []string
	typeStr := this.EffectiveTypeString()

	// Check for swaggertype tag override - takes precedence over automatic type inference
	tags := this.GetTags()
	if swaggerType, ok := tags["swaggertype"]; ok {
		// Parse comma-separated type keywords
		typeKeywords := strings.Split(swaggerType, ",")
		for i := range typeKeywords {
			typeKeywords[i] = strings.TrimSpace(typeKeywords[i])
		}

		// Build custom schema using existing schemautil package function
		baseSchema, err := schemautil.BuildCustomSchema(typeKeywords)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid swaggertype tag '%s' for field %s: %w", swaggerType, this.Name, err)
		}

		// Apply other struct tags to enrich the schema
		if err := this.applyStructTagsToSchema(baseSchema); err != nil {
			return nil, nil, fmt.Errorf("failed to apply tags to schema for field %s: %w", this.Name, err)
		}

		return baseSchema, nil, nil
	}

	var debug bool
	if strings.Contains(typeStr, "constants.") {
		debug = true
	}
	if debug {
		console.Logger.Debug("Building schema for type: $Bold{%s} (TypeString: $Bold{%s})\n", typeStr, this.TypeString)
	}

	// Save full type string before normalization for accurate $ref creation.
	fullTypeStr := typeStr

	// Normalize type name to short form
	typeStr = normalizeTypeName(typeStr)
	if debug && typeStr != this.TypeString {
		console.Logger.Debug("Normalized type name to: $Bold{%s}\n", typeStr)
	}

	// Remove pointer prefix
	isPointer := strings.HasPrefix(typeStr, "*")
	if isPointer {
		typeStr = strings.TrimPrefix(typeStr, "*")
	}
	fullTypeStr = strings.TrimPrefix(fullTypeStr, "*")

	// Create a normalized field for method-based type checks
	normalizedField := &StructField{TypeString: typeStr}

	// Handle any/interface{} types as empty schema (unknown/any value)
	if normalizedField.IsAny() {
		if debug {
			console.Logger.Debug("Detected any/interface{} type: $Bold{%s}\n", typeStr)
		}
		return &spec.Schema{}, nil, nil
	}

	// Check if this is a fields wrapper type
	if normalizedField.IsFieldsWrapper() {
		if debug {
			console.Logger.Debug("Detected fields wrapper type: $Bold{%s}\n", typeStr)
		}
		schema, nestedTypes, err := getPrimitiveSchemaForFieldType(typeStr, this.TypeString, enumLookup)
		if err != nil {
			return nil, nil, err
		}
		// Apply struct tags to enrich the schema
		if err := this.applyStructTagsToSchema(schema); err != nil {
			return nil, nil, fmt.Errorf("failed to apply tags to schema: %w", err)
		}
		return schema, nestedTypes, nil
	}

	// Handle primitive types
	if normalizedField.IsPrimitive() {
		schema := primitiveTypeToSchema(typeStr)
		if debug {
			console.Logger.Debug("Detected Is Primitive type: $Bold{%s} Schema %+v\n", typeStr, schema)
		}
		// Apply struct tags to enrich the schema
		if err := this.applyStructTagsToSchema(schema); err != nil {
			return nil, nil, fmt.Errorf("failed to apply tags to schema: %w", err)
		}
		return schema, nil, nil
	}

	// Handle arrays — create child StructField for element type
	if strings.HasPrefix(typeStr, "[]") {
		elemType := strings.TrimPrefix(typeStr, "[]")
		fullElemType := elemType
		if strings.HasPrefix(fullTypeStr, "[]") {
			fullElemType = strings.TrimPrefix(fullTypeStr, "[]")
		}
		elemField := &StructField{TypeString: fullElemType}
		elemSchema, elemNestedTypes, err := elemField.BuildSchema(public, forceRequired, enumLookup)
		if err != nil {
			return nil, nil, err
		}
		schema := spec.ArrayProperty(elemSchema)
		// Apply struct tags to enrich the schema
		if err := this.applyStructTagsToSchema(schema); err != nil {
			return nil, nil, fmt.Errorf("failed to apply tags to schema: %w", err)
		}
		return schema, elemNestedTypes, nil
	}

	// Handle maps — create child StructField for value type
	if strings.HasPrefix(typeStr, "map[") {
		bracketCount := 0
		valueStart := -1
		for i, ch := range typeStr {
			if ch == '[' {
				bracketCount++
			} else if ch == ']' {
				bracketCount--
				if bracketCount == 0 {
					valueStart = i + 1
					break
				}
			}
		}
		if valueStart == -1 {
			return nil, nil, fmt.Errorf("invalid map type: %s", typeStr)
		}
		fullValueType := typeStr[valueStart:]
		if strings.HasPrefix(fullTypeStr, "map[") {
			fullBracketCount := 0
			fullValueStart := -1
			for i, ch := range fullTypeStr {
				if ch == '[' {
					fullBracketCount++
				} else if ch == ']' {
					fullBracketCount--
					if fullBracketCount == 0 {
						fullValueStart = i + 1
						break
					}
				}
			}
			if fullValueStart != -1 {
				fullValueType = fullTypeStr[fullValueStart:]
			}
		}
		valueField := &StructField{TypeString: fullValueType}
		valueSchema, valueNestedTypes, err := valueField.BuildSchema(public, forceRequired, enumLookup)
		if err != nil {
			return nil, nil, err
		}
		schema := spec.MapProperty(valueSchema)
		// Apply struct tags to enrich the schema
		if err := this.applyStructTagsToSchema(schema); err != nil {
			return nil, nil, fmt.Errorf("failed to apply tags to schema: %w", err)
		}
		return schema, valueNestedTypes, nil
	}

	// Check if this is an enum type
	if enumLookup != nil {
		if debug {
			console.Logger.Debug("Checking enum for type: $Bold{%s}\n", typeStr)
		}
		enums, err := enumLookup.GetEnumsForType(typeStr, nil)
		if err == nil && len(enums) > 0 {
			if debug {
				console.Logger.Debug("Detected Enum type: $Bold{%s} with %d values, creating $ref\n", typeStr, len(enums))
			}
			refName := resolveRefName(typeStr, fullTypeStr)
			schema := spec.RefSchema("#/definitions/" + refName)
			// Propagate full import path for correct cross-package resolution
			if strings.Contains(fullTypeStr, "/") {
				nestedTypes = append(nestedTypes, fullTypeStr)
			} else {
				nestedTypes = append(nestedTypes, refName)
			}
			return schema, nestedTypes, nil
		}
		if debug {
			if err != nil {
				console.Logger.Debug("Error looking up enums for type: $Bold{%s}: $Red{%s}\n", typeStr, err.Error())
			}
		}
	} else {
		if debug {
			console.Logger.Debug("No enumLookup provided, skipping enum check for type: $Bold{%s}\n", typeStr)
		}
	}

	// Struct ref — validate brackets
	typeName := typeStr
	bracketDepth := 0
	for _, ch := range typeName {
		if ch == '[' {
			bracketDepth++
		} else if ch == ']' {
			bracketDepth--
		}
	}
	if bracketDepth != 0 {
		console.Logger.Debug("Skipping reference creation for malformed type name with unbalanced brackets: %s\n", typeName)
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}, nil, nil
	}

	// Guard against malformed type names where struct tags leak into the type string
	// (e.g., anonymous struct fields). These would produce URL-encoded $ref names.
	if strings.ContainsAny(typeName, " \t\"'`=:;") {
		console.Logger.Debug("Skipping ref for malformed type name: %s\n", typeName)
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}, nil, nil
	}

	refName := resolveRefName(typeName, fullTypeStr)
	if public {
		refName = refName + "Public"
	}

	schema := spec.RefSchema("#/definitions/" + refName)
	// Propagate full import path for correct cross-package resolution.
	// When fullTypeStr lacks a "/" (short form like "global_struct.EventProperties"),
	// try to recover the full import path from the go/types Type field so that
	// buildSchemasRecursive can locate the correct package instead of guessing
	// a sibling path. Only use the resolved path if its short form matches typeStr
	// to avoid using wrapper types (e.g., fields.StructField) for the inner type.
	nestedFullPath := fullTypeStr
	if !strings.Contains(nestedFullPath, "/") && this.Type != nil {
		if resolved := resolveFullImportPath(this.Type); resolved != "" {
			resolvedShort := normalizeTypeName(resolved)
			if resolvedShort == typeName {
				nestedFullPath = resolved
			}
		}
	}
	if strings.Contains(nestedFullPath, "/") {
		nestedRef := nestedFullPath
		if public {
			nestedRef += "Public"
		}
		nestedTypes = append(nestedTypes, nestedRef)
	} else {
		nestedTypes = append(nestedTypes, refName)
	}
	if debug {
		console.Logger.Debug("Created Ref Schema for type: $Bold{$Red{%s}} Ref: $Bold{#/definitions/%s}\n", typeStr, refName)
	}
	return schema, nestedTypes, nil
}

// applyStructTagsToSchema enriches a base schema with metadata from struct tags.
// Handles enums, format, title, constraints (min/max, minLength/maxLength),
// default, example, readonly, multipleOf, and extensions tags.
// The base schema's type structure should already be set before calling this.
func (this *StructField) applyStructTagsToSchema(schema *spec.Schema) error {
	if schema == nil {
		return nil
	}

	tags := this.GetTags()

	// Apply format tag
	if format, ok := tags["format"]; ok {
		schema.Format = format
	}

	// Apply title tag
	if title, ok := tags["title"]; ok {
		schema.Title = title
	}

	// Apply enums tag
	if enumsStr, ok := tags["enums"]; ok {
		// Split by comma, handling escaped UTF-8 commas (0x2C)
		enumValues := strings.Split(enumsStr, ",")
		var enums []interface{}

		// Determine target type for enum parsing
		targetType := ""
		if len(schema.Type) > 0 {
			targetType = schema.Type[0]
			// For array types, check the item type
			if targetType == "array" && schema.Items != nil && schema.Items.Schema != nil && len(schema.Items.Schema.Type) > 0 {
				targetType = schema.Items.Schema.Type[0]
			}
		}

		for _, val := range enumValues {
			val = strings.TrimSpace(val)
			// Replace UTF-8 hex comma back to actual comma
			val = strings.ReplaceAll(val, "0x2C", ",")
			if val != "" {
				// Try to parse as number if target type is integer/number
				if strings.Contains(targetType, "integer") || strings.Contains(targetType, "number") {
					if num, err := strconv.ParseFloat(val, 64); err == nil {
						if strings.Contains(targetType, "integer") {
							enums = append(enums, int(num))
						} else {
							enums = append(enums, num)
						}
						continue
					}
				}
				enums = append(enums, val)
			}
		}
		schema.Enum = enums

		// Apply x-enum-varnames extension if present
		if varNames, ok := tags["x-enum-varnames"]; ok {
			if schema.Extensions == nil {
				schema.Extensions = make(spec.Extensions)
			}
			varNamesList := strings.Split(varNames, ",")
			for i := range varNamesList {
				varNamesList[i] = strings.TrimSpace(varNamesList[i])
			}
			schema.Extensions["x-enum-varnames"] = varNamesList
		}
	}

	// Apply minimum/maximum constraints
	if minStr, ok := tags["minimum"]; ok {
		if min, err := strconv.ParseFloat(minStr, 64); err == nil {
			schema.Minimum = &min
		}
	}
	if maxStr, ok := tags["maximum"]; ok {
		if max, err := strconv.ParseFloat(maxStr, 64); err == nil {
			schema.Maximum = &max
		}
	}

	// Apply minLength/maxLength constraints
	if minLenStr, ok := tags["minLength"]; ok {
		if minLen, err := strconv.ParseInt(minLenStr, 10, 64); err == nil {
			schema.MinLength = &minLen
		}
	}
	if maxLenStr, ok := tags["maxLength"]; ok {
		if maxLen, err := strconv.ParseInt(maxLenStr, 10, 64); err == nil {
			schema.MaxLength = &maxLen
		}
	}

	// Apply default value
	if defaultStr, ok := tags["swag_default"]; ok {
		// Try to parse as JSON, fallback to string
		var defaultVal interface{}
		if err := json.Unmarshal([]byte(defaultStr), &defaultVal); err == nil {
			schema.Default = defaultVal
		} else {
			schema.Default = defaultStr
		}
	}

	// Apply example value
	if exampleStr, ok := tags["example"]; ok {
		// Try to parse as JSON, fallback to string
		var exampleVal interface{}
		if err := json.Unmarshal([]byte(exampleStr), &exampleVal); err == nil {
			schema.Example = exampleVal
		} else {
			schema.Example = exampleStr
		}
	}

	// Apply readonly
	if readonlyStr, ok := tags["readonly"]; ok {
		if readonly, err := strconv.ParseBool(readonlyStr); err == nil {
			schema.ReadOnly = readonly
		}
	}

	// Apply multipleOf
	if multipleStr, ok := tags["multipleOf"]; ok {
		if multiple, err := strconv.ParseFloat(multipleStr, 64); err == nil {
			schema.MultipleOf = &multiple
		}
	}

	// Apply custom extensions
	if extensionsStr, ok := tags["extensions"]; ok {
		if schema.Extensions == nil {
			schema.Extensions = make(spec.Extensions)
		}
		// Format: "x-key1:value1,x-key2:value2"
		extPairs := strings.Split(extensionsStr, ",")
		for _, pair := range extPairs {
			parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				// Try to parse value as JSON
				var jsonVal interface{}
				if err := json.Unmarshal([]byte(val), &jsonVal); err == nil {
					schema.Extensions[key] = jsonVal
				} else {
					schema.Extensions[key] = val
				}
			}
		}
	}

	return nil
}

// TypeEnumLookup is an interface for looking up enum values for a type
type TypeEnumLookup interface {
	GetEnumsForType(typeName string, file *ast.File) ([]EnumValue, error)
}

// EnumValue represents an enum constant
type EnumValue struct {
	Key     string
	Value   interface{}
	Comment string
}

// DefinitionNameResolver resolves the canonical definition name for a type.
// For unique types it returns the short name (e.g., "constants.Role").
// For NotUnique types it returns the full-path name
// (e.g., "github_com_chargebee_chargebee-go_v3_enum.Source").
type DefinitionNameResolver interface {
	ResolveDefinitionName(fullTypePath string) string
}

// resolveFullImportPath extracts the full import path from a go/types Type.
// Returns "pkgPath.TypeName" (e.g., "github.com/.../global_struct.EventProperties")
// or empty string if the path cannot be determined.
// Unwraps pointers and slices to find the underlying named type.
func resolveFullImportPath(t types.Type) string {
	if t == nil {
		return ""
	}
	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	// Unwrap slice
	if sl, ok := t.(*types.Slice); ok {
		t = sl.Elem()
		// Unwrap pointer inside slice
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
	}
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return ""
	}
	return pkg.Path() + "." + named.Obj().Name()
}

// globalNameResolver is the active definition name resolver set by the
// orchestrator before schema building. When nil, buildSchemaForType falls
// back to makeFullPathDefinitionName (backward-compatible default).
var globalNameResolver DefinitionNameResolver

// SetGlobalNameResolver sets the global definition name resolver.
func SetGlobalNameResolver(r DefinitionNameResolver) {
	globalNameResolver = r
}

// resolveRefName resolves the $ref name for a type. When a global name
// resolver is configured it returns the canonical name (short for unique
// types, full-path for NotUnique). Otherwise it falls back to
// makeFullPathDefinitionName for backward compatibility.
func resolveRefName(shortName, fullPath string) string {
	if !strings.Contains(fullPath, "/") {
		return shortName
	}
	if globalNameResolver != nil {
		return globalNameResolver.ResolveDefinitionName(fullPath)
	}
	return makeFullPathDefinitionName(fullPath)
}

// ToSpecSchema converts a StructField to OpenAPI spec.Schema
// propName: extracted from json tag (first part before comma)
// schema: the OpenAPI schema for this field
// required: true if omitempty is absent from json tag (or forceRequired is true)
// nestedTypes: list of struct type names encountered for recursive definition generation
// forceRequired: if true, field is always required regardless of omitempty tag
func (this *StructField) ToSpecSchema(
	public bool,
	forceRequired bool,
	enumLookup TypeEnumLookup,
) (propName string, schema *spec.Schema, required bool, nestedTypes []string, err error) {
	// Filter field if public mode and field is not public
	if public && !this.IsPublic() {
		return "", nil, false, nil, nil
	}

	// Check for swaggerignore tag
	tags := this.GetTags()
	if swaggerIgnore, ok := tags["swaggerignore"]; ok && strings.EqualFold(swaggerIgnore, "true") {
		console.Logger.Debug("$Red{$Bold{Ignoring field %s due to swaggerignore tag}}\n", this.Name)
		return "", nil, false, nil, nil
	}

	// Extract property name from json tag
	jsonTag := tags["json"]
	if jsonTag == "" {
		jsonTag = tags["column"]
	}
	if jsonTag == "" {
		return "", nil, false, nil, nil
	}

	parts := strings.Split(jsonTag, ",")
	propName = parts[0]

	// Check for omitempty to determine required
	if forceRequired {
		required = true
	} else {
		required = true
		for _, part := range parts[1:] {
			if strings.TrimSpace(part) == "omitempty" {
				required = false
				break
			}
		}
	}

	// Skip if json tag is "-"
	if propName == "-" {
		return "", nil, false, nil, nil
	}

	// Resolve the effective type string for schema building
	// For generic wrappers, extract the type parameter and build schema from that
	if this.IsGeneric() {
		extractedType, extractErr := this.GenericTypeArg()
		if extractErr != nil {
			return "", nil, false, nil, fmt.Errorf("failed to extract type parameter from %s: %w", this.EffectiveTypeString(), extractErr)
		}

		// Determine effective public: only struct type args get Public suffix
		effectivePublic := public
		if public && this.Type != nil {
			if !this.IsGenericTypeArgStruct() {
				effectivePublic = false
			}
		}

		schemaField := &StructField{TypeString: extractedType, Type: this.Type}
		schema, nestedTypes, err = schemaField.BuildSchema(effectivePublic, forceRequired, enumLookup)
	} else {
		// Determine effective public: only struct types get Public suffix
		effectivePublic := public
		if public && this.Type != nil {
			if !this.IsUnderlyingStruct() {
				effectivePublic = false
			}
		}

		schema, nestedTypes, err = this.BuildSchema(effectivePublic, forceRequired, enumLookup)
	}

	if err != nil {
		return "", nil, false, nil, fmt.Errorf("failed to build schema for type %s: %w", this.EffectiveTypeString(), err)
	}

	return propName, schema, required, nestedTypes, nil
}

// normalizeTypeName converts a full module path type name to short form
// e.g., "github.com/griffnb/core-swag/testing/testdata/core_models/constants.UnionStatus" -> "constants.UnionStatus"
// Handles full paths, short names, pointer types, and array prefixes
func normalizeTypeName(typeStr string) string {
	// Strip prefix modifiers (pointer, slice) before normalization, re-add after
	prefix := ""
	for strings.HasPrefix(typeStr, "[]") || strings.HasPrefix(typeStr, "*") {
		if strings.HasPrefix(typeStr, "[]") {
			prefix += "[]"
			typeStr = typeStr[2:]
		} else {
			prefix += "*"
			typeStr = typeStr[1:]
		}
	}

	if strings.Contains(typeStr, "/") {
		// For generic types like "github.com/.../fields.IntConstantField[github.com/.../constants.Role]",
		// normalize the base type and each type parameter separately.
		if bracketIdx := strings.Index(typeStr, "["); bracketIdx >= 0 {
			basePart := typeStr[:bracketIdx]
			paramPart := typeStr[bracketIdx:] // "[github.com/.../constants.Role]"

			// Normalize base
			if lastSlash := strings.LastIndex(basePart, "/"); lastSlash >= 0 {
				basePart = basePart[lastSlash+1:]
			}

			// Normalize type parameters inside brackets
			inner := paramPart[1 : len(paramPart)-1] // strip [ and ]
			inner = normalizeTypeName(inner)
			typeStr = basePart + "[" + inner + "]"
		} else {
			// Non-generic: find the last slash and take everything after it
			lastSlash := strings.LastIndex(typeStr, "/")
			if lastSlash >= 0 {
				typeStr = typeStr[lastSlash+1:]
			}
		}
	}

	return prefix + typeStr
}

// isGenericTypeArgStruct checks whether the first type argument of a generic type
// is a struct. Returns true if the type has no type arguments (not a generic),
// or if the argument's underlying Go type is a struct.
// Returns false if the argument's underlying type is a primitive (e.g., int, string),
// meaning it's an enum or type alias that should not get a Public suffix.
func isGenericTypeArgStruct(t types.Type) bool {
	if t == nil {
		return true
	}

	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		return isGenericTypeArgStruct(ptr.Elem())
	}

	named, ok := t.(*types.Named)
	if !ok {
		return true
	}

	// Check if this is a generic instantiation with type arguments
	typeArgs := named.TypeArgs()
	if typeArgs == nil || typeArgs.Len() == 0 {
		return true
	}

	// Get the first type argument and check its underlying type
	arg := typeArgs.At(0)
	_, isStruct := arg.Underlying().(*types.Struct)
	return isStruct
}

// isUnderlyingStruct checks whether a type's underlying Go type is a struct.
// Unwraps pointers and named types. Returns true for unknown types (safe default).
func isUnderlyingStruct(t types.Type) bool {
	if t == nil {
		return true
	}

	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		return isUnderlyingStruct(ptr.Elem())
	}

	named, ok := t.(*types.Named)
	if !ok {
		return true
	}

	_, isStruct := named.Underlying().(*types.Struct)
	return isStruct
}

// extractGenericTypeParameter extracts the type parameter from any generic type
// Handles patterns like StructField[T], IntConstantField[T], StringField[T], etc.
// Also handles nested brackets like Field[map[string][]User]
func extractGenericTypeParameter(typeStr string) (string, error) {
	// Find the opening bracket
	idx := strings.Index(typeStr, "[")
	if idx == -1 {
		return "", fmt.Errorf("opening bracket [ not found in %s", typeStr)
	}

	// Start after "["
	start := idx + 1
	bracketCount := 1
	end := start

	// Count brackets to find matching closing bracket
	for end < len(typeStr) && bracketCount > 0 {
		switch typeStr[end] {
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}
		if bracketCount > 0 {
			end++
		}
	}

	if bracketCount != 0 {
		return "", fmt.Errorf("mismatched brackets in %s", typeStr)
	}

	extracted := typeStr[start:end]

	// Remove leading * if it's a pointer
	extracted = strings.TrimPrefix(extracted, "*")

	return extracted, nil
}

// makeFullPathDefinitionName converts a full module path type string to the
// definition name format used by TypeSpecDef.TypeName() for NotUnique types.
// Input:  "github.com/chargebee/chargebee-go/v3/enum.Source"
// Output: "github_com_chargebee_chargebee-go_v3_enum.Source"
// This matches the algorithm in TypeSpecDef.TypeName() at internal/domain/types.go:62-67.
func makeFullPathDefinitionName(fullTypeStr string) string {
	// Split at last "." to separate package path from type name
	lastDot := strings.LastIndex(fullTypeStr, ".")
	if lastDot == -1 {
		return fullTypeStr
	}
	pkgPath := fullTypeStr[:lastDot]
	typeName := fullTypeStr[lastDot+1:]

	// Replace \, /, . in pkgPath with _ (same transform as TypeSpecDef.TypeName)
	pkgPath = strings.Map(func(r rune) rune {
		if r == '\\' || r == '/' || r == '.' {
			return '_'
		}
		return r
	}, pkgPath)

	return pkgPath + "." + typeName
}

// getPrimitiveSchemaForFieldType returns the appropriate schema for a fields wrapper type
func getPrimitiveSchemaForFieldType(typeStr string, originalTypeStr string, enumLookup TypeEnumLookup) (*spec.Schema, []string, error) {
	// Check for IntConstantField and StringConstantField with enum type parameters
	// Create $ref to enum definition instead of inlining enum values
	if strings.Contains(typeStr, "fields.IntConstantField[") {
		enumType := extractConstantFieldEnumType(originalTypeStr)
		if enumType != "" {
			normalizedEnum := normalizeTypeName(enumType)
			if enumLookup != nil {
				enums, err := enumLookup.GetEnumsForType(normalizedEnum, nil)
				if err == nil && len(enums) > 0 {
					refName := resolveRefName(normalizedEnum, enumType)
					schema := spec.RefSchema("#/definitions/" + refName)
					// Propagate full import path for correct cross-package resolution
					if strings.Contains(enumType, "/") {
						return schema, []string{enumType}, nil
					}
					return schema, []string{refName}, nil
				}
			}
		}
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.StringConstantField[") {
		enumType := extractConstantFieldEnumType(originalTypeStr)
		if enumType != "" {
			normalizedEnum := normalizeTypeName(enumType)
			if enumLookup != nil {
				enums, err := enumLookup.GetEnumsForType(normalizedEnum, nil)
				if err == nil && len(enums) > 0 {
					refName := resolveRefName(normalizedEnum, enumType)
					schema := spec.RefSchema("#/definitions/" + refName)
					// Propagate full import path for correct cross-package resolution
					if strings.Contains(enumType, "/") {
						return schema, []string{enumType}, nil
					}
					return schema, []string{refName}, nil
				}
			}
		}
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.StringField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.IntField") || strings.Contains(typeStr, "fields.DecimalField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.UUIDField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "uuid"}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.BoolField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"boolean"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.FloatField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}}}, nil, nil
	}
	if strings.Contains(typeStr, "fields.TimeField") {
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "date-time"}}, nil, nil
	}
	// Default to string for unknown field types
	return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}, nil, nil
}

// extractConstantFieldEnumType extracts the enum type from IntConstantField[T] or StringConstantField[T]
func extractConstantFieldEnumType(typeStr string) string {
	// Look for pattern like "*fields.IntConstantField[constants.Role]" or "*fields.StringConstantField[constants.GlobalConfigKey]"
	if strings.Contains(typeStr, "ConstantField[") {
		start := strings.Index(typeStr, "[")
		end := strings.LastIndex(typeStr, "]")
		if start != -1 && end != -1 && end > start {
			return typeStr[start+1 : end]
		}
	}
	return ""
}

// applyEnumsToSchema applies enum values to a schema
func applyEnumsToSchema(schema *spec.Schema, enums []EnumValue) {
	if len(enums) == 0 {
		return
	}

	var enumValues []interface{}
	var varNames []string
	enumComments := make(map[string]string)
	var enumDescriptions []string

	dedupeMap := make(map[interface{}]bool)

	for _, enum := range enums {
		if _, exists := dedupeMap[enum.Value]; exists {
			continue
		}
		dedupeMap[enum.Value] = true
		enumValues = append(enumValues, enum.Value)
		varNames = append(varNames, enum.Key)
		enumDescriptions = append(enumDescriptions, enum.Comment)
		if enum.Comment != "" {
			enumComments[enum.Key] = enum.Comment
		}
	}

	schema.Enum = enumValues

	if schema.Extensions == nil {
		schema.Extensions = make(spec.Extensions)
	}
	schema.Extensions["x-enum-varnames"] = varNames

	if len(enumComments) > 0 {
		schema.Extensions["x-enum-comments"] = enumComments
		schema.Extensions["x-enum-descriptions"] = enumDescriptions
	}
}

// primitiveTypeToSchema converts a Go primitive type to OpenAPI schema
func primitiveTypeToSchema(typeStr string) *spec.Schema {
	switch typeStr {
	case "string":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}
	case "bool":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"boolean"}}}
	case "int", "uint":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}}}
	case "int8", "uint8", "int16", "uint16", "int32", "uint32", "byte", "rune":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}, Format: "int32"}}
	case "int64", "uint64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"integer"}, Format: "int64"}}
	case "float32":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}, Format: "float"}}
	case "float64":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}, Format: "double"}}
	case "time.Time", "*time.Time":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "date-time"}}
	case "types.UUID", "*types.UUID", "uuid.UUID", "*uuid.UUID",
		"github.com/griffnb/core/lib/types.UUID", "*github.com/griffnb/core/lib/types.UUID",
		"github.com/google/uuid.UUID", "*github.com/google/uuid.UUID":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "uuid"}}
	case "decimal.Decimal", "*decimal.Decimal", "github.com/shopspring/decimal.Decimal", "*github.com/shopspring/decimal.Decimal":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"number"}}}
	case "json.RawMessage", "encoding/json.RawMessage":
		// json.RawMessage is []byte representing arbitrary JSON — any valid JSON value
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}
	case "[]byte", "[]uint8":
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}, Format: "byte"}}
	default:
		return &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{typeStr}}}
	}
}
