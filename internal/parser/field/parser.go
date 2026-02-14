package field

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"

	"github.com/go-openapi/spec"
)

// FieldParser is the interface for parsing struct fields.
type FieldParser interface {
	// ShouldSkip determines if the field should be skipped
	ShouldSkip() bool

	// FieldNames returns the JSON field names for this field
	FieldNames() ([]string, error)

	// FirstTagValue returns the first value from a tag
	FirstTagValue(tag string) string

	// FormName returns the form parameter name
	FormName() string

	// HeaderName returns the header parameter name
	HeaderName() string

	// PathName returns the path parameter name
	PathName() string

	// CustomSchema returns a custom schema if swaggertype tag is present
	CustomSchema() (*spec.Schema, error)

	// ComplementSchema complements the schema with field properties from tags
	ComplementSchema(schema *spec.Schema) error

	// IsRequired determines if the field is required
	IsRequired() (bool, error)
}

// TagBaseFieldParser parses struct field using tags.
type TagBaseFieldParser struct {
	schemaHelper SchemaHelper
	config       ParserConfig
	field        *ast.Field
	tag          reflect.StructTag
}

// NewTagBaseFieldParser creates a new tag-based field parser.
func NewTagBaseFieldParser(
	schemaHelper SchemaHelper,
	config ParserConfig,
	field *ast.Field,
) FieldParser {
	parser := &TagBaseFieldParser{
		schemaHelper: schemaHelper,
		config:       config,
		field:        field,
		tag:          "",
	}
	if parser.field.Tag != nil {
		parser.tag = reflect.StructTag(strings.ReplaceAll(field.Tag.Value, "`", ""))
	}

	return parser
}

// ShouldSkip determines if the field should be skipped.
func (ps *TagBaseFieldParser) ShouldSkip() bool {
	// Skip non-exported fields.
	if ps.field.Names != nil && !ast.IsExported(ps.field.Names[0].Name) {
		return true
	}

	if ps.field.Tag == nil {
		return false
	}

	ignoreTag := ps.tag.Get(swaggerIgnoreTag)
	if strings.EqualFold(ignoreTag, "true") {
		return true
	}

	// json:"tag,hoge"
	name := strings.TrimSpace(strings.Split(ps.tag.Get(jsonTag), ",")[0])
	if name == "-" {
		return true
	}

	return false
}

// FieldNames returns the JSON field names for this field.
func (ps *TagBaseFieldParser) FieldNames() ([]string, error) {
	if len(ps.field.Names) <= 1 {
		// if embedded but with a json/form name
		if ps.field.Tag != nil {
			// json:"tag,hoge"
			name := strings.TrimSpace(strings.Split(ps.tag.Get(jsonTag), ",")[0])
			if name != "" {
				return []string{name}, nil
			}

			// use "form" tag over json tag
			name = ps.FormName()
			if name != "" {
				return []string{name}, nil
			}
		}
		if len(ps.field.Names) == 0 {
			return nil, nil
		}
	}
	var names = make([]string, 0, len(ps.field.Names))
	for _, name := range ps.field.Names {
		names = append(names, ApplyNamingStrategy(name.Name, ps.config.GetNamingStrategy()))
	}
	return names, nil
}

// FirstTagValue returns the first value from a tag.
func (ps *TagBaseFieldParser) FirstTagValue(tag string) string {
	if ps.field.Tag != nil {
		return strings.TrimRight(strings.TrimSpace(strings.Split(ps.tag.Get(tag), ",")[0]), "[]")
	}
	return ""
}

// FormName returns the form parameter name.
func (ps *TagBaseFieldParser) FormName() string {
	return ps.FirstTagValue(formTag)
}

// HeaderName returns the header parameter name.
func (ps *TagBaseFieldParser) HeaderName() string {
	return ps.FirstTagValue(headerTag)
}

// PathName returns the path parameter name.
func (ps *TagBaseFieldParser) PathName() string {
	return ps.FirstTagValue(uriTag)
}

// CustomSchema returns a custom schema if swaggertype tag is present.
func (ps *TagBaseFieldParser) CustomSchema() (*spec.Schema, error) {
	if ps.field.Tag == nil {
		return nil, nil
	}

	typeTag := ps.tag.Get(swaggerTypeTag)
	if typeTag != "" {
		return ps.schemaHelper.BuildCustomSchema(strings.Split(typeTag, ","))
	}

	return nil, nil
}

// IsRequired determines if the field is required.
func (ps *TagBaseFieldParser) IsRequired() (bool, error) {
	if ps.field.Tag == nil {
		// No tags at all - use RequiredByDefault
		return ps.config.IsRequiredByDefault(), nil
	}

	bindingTag := ps.tag.Get(bindingTag)
	if bindingTag != "" {
		for _, val := range strings.Split(bindingTag, ",") {
			switch val {
			case requiredLabel:
				return true, nil
			case optionalLabel:
				return false, nil
			}
		}
	}

	validateTag := ps.tag.Get(validateTag)
	if validateTag != "" {
		for _, val := range strings.Split(validateTag, ",") {
			switch val {
			case requiredLabel:
				return true, nil
			case optionalLabel:
				return false, nil
			}
		}
	}

	jsonTag := ps.tag.Get(jsonTag)
	if jsonTag != "" {
		for _, val := range strings.Split(jsonTag, ",") {
			if val == omitEmptyLabel {
				return false, nil
			}
		}
	}

	return ps.config.IsRequiredByDefault(), nil
}

// ComplementSchema complements the schema with field properties from tags.
func (ps *TagBaseFieldParser) ComplementSchema(schema *spec.Schema) error {
	types := ps.schemaHelper.GetSchemaTypePath(schema, 2)
	if len(types) == 0 {
		return fmt.Errorf("invalid type for field: %s", ps.field.Names[0])
	}

	if ps.schemaHelper.IsRefSchema(schema) {
		var newSchema = spec.Schema{}
		err := ps.complementSchema(&newSchema, types)
		if err != nil {
			return err
		}
		if !reflect.ValueOf(newSchema).IsZero() {
			*schema = *(newSchema.WithAllOf(*schema))
		}
		return nil
	}

	return ps.complementSchema(schema, types)
}

// complementSchema is the internal implementation that complements a schema.
func (ps *TagBaseFieldParser) complementSchema(schema *spec.Schema, types []string) error {
	if ps.field.Tag == nil {
		if ps.field.Doc != nil {
			schema.Description = strings.TrimSpace(ps.field.Doc.Text())
		}

		if schema.Description == "" && ps.field.Comment != nil {
			schema.Description = strings.TrimSpace(ps.field.Comment.Text())
		}

		return nil
	}

	field := &structField{
		schemaType: types[0],
		formatType: ps.tag.Get(formatTag),
		title:      ps.tag.Get(titleTag),
	}

	if len(types) > 1 && (types[0] == ARRAY || types[0] == OBJECT) {
		field.arrayType = types[1]
	}

	jsonTagValue := ps.tag.Get(jsonTag)

	bindingTagValue := ps.tag.Get(bindingTag)
	if bindingTagValue != "" {
		parseValidTags(bindingTagValue, field)
	}

	validateTagValue := ps.tag.Get(validateTag)
	if validateTagValue != "" {
		parseValidTags(validateTagValue, field)
	}

	// Convert oneof enums from strings to proper types (set by parseValidTags)
	if len(field.enums) > 0 {
		enumType := field.schemaType
		if field.schemaType == ARRAY {
			enumType = field.arrayType
		}

		convertedEnums := make([]interface{}, 0, len(field.enums))
		for _, e := range field.enums {
			// Convert string enum values to proper type
			strValue, ok := e.(string)
			if !ok {
				convertedEnums = append(convertedEnums, e)
				continue
			}
			value, err := ps.schemaHelper.DefineType(enumType, strValue)
			if err != nil {
				// Skip invalid values
				continue
			}
			convertedEnums = append(convertedEnums, value)
		}
		field.enums = convertedEnums
	}

	enumsTagValue := ps.tag.Get(enumsTag)
	if enumsTagValue != "" {
		err := ps.parseEnumTags(enumsTagValue, field)
		if err != nil {
			return err
		}
	}

	if ps.schemaHelper.IsNumericType(field.schemaType) || ps.schemaHelper.IsNumericType(field.arrayType) {
		maximum, err := getFloatTag(ps.tag, maximumTag)
		if err != nil {
			return err
		}

		if maximum != nil {
			field.maximum = maximum
		}

		minimum, err := getFloatTag(ps.tag, minimumTag)
		if err != nil {
			return err
		}

		if minimum != nil {
			field.minimum = minimum
		}

		multipleOf, err := getFloatTag(ps.tag, multipleOfTag)
		if err != nil {
			return err
		}

		if multipleOf != nil {
			field.multipleOf = multipleOf
		}
	}

	if field.schemaType == STRING || field.arrayType == STRING {
		maxLength, err := getIntTag(ps.tag, maxLengthTag)
		if err != nil {
			return err
		}

		if maxLength != nil {
			field.maxLength = maxLength
		}

		minLength, err := getIntTag(ps.tag, minLengthTag)
		if err != nil {
			return err
		}

		if minLength != nil {
			field.minLength = minLength
		}
	}

	// json:"name,string" or json:",string"
	exampleTagValue, ok := ps.tag.Lookup(exampleTag)
	if ok {
		field.exampleValue = exampleTagValue

		if !strings.Contains(jsonTagValue, ",string") {
			example, err := ps.schemaHelper.DefineTypeOfExample(field.schemaType, field.arrayType, exampleTagValue)
			if err != nil {
				return err
			}

			field.exampleValue = example
		}
	}

	// perform this after setting everything else (min, max, etc...)
	if strings.Contains(jsonTagValue, ",string") {
		// @encoding/json: "It applies only to fields of string, floating point, integer, or boolean types."
		defaultValues := map[string]string{
			// Zero Values as string
			STRING:  "",
			INTEGER: "0",
			BOOLEAN: "false",
			NUMBER:  "0",
		}

		defaultValue, ok := defaultValues[field.schemaType]
		if ok {
			field.schemaType = STRING
			*schema = *ps.schemaHelper.PrimitiveSchema(field.schemaType)

			if field.exampleValue == nil {
				// if exampleValue is not defined by the user,
				// we will force an example with a correct value
				// (eg: int->"0", bool:"false")
				field.exampleValue = defaultValue
			}
		}
	}

	if ps.field.Doc != nil {
		schema.Description = strings.TrimSpace(ps.field.Doc.Text())
	}

	if schema.Description == "" && ps.field.Comment != nil {
		schema.Description = strings.TrimSpace(ps.field.Comment.Text())
	}

	schema.ReadOnly = ps.tag.Get(readOnlyTag) == "true"

	defaultTagValue, ok := ps.tag.Lookup(defaultTag)
	if ok {
		value, err := ps.schemaHelper.DefineType(field.schemaType, defaultTagValue)
		if err != nil {
			return err
		}

		schema.Default = value
	}

	schema.Example = field.exampleValue

	if field.schemaType != ARRAY {
		schema.Format = field.formatType
	}
	schema.Title = field.title

	extensionsTagValue := ps.tag.Get(extensionsTag)
	if extensionsTagValue != "" {
		schema.Extensions = ps.schemaHelper.SetExtensionParam(extensionsTagValue)
	}

	varNamesTag := ps.tag.Get("x-enum-varnames")
	if varNamesTag != "" {
		varNames := strings.Split(varNamesTag, ",")
		if len(varNames) != len(field.enums) {
			return fmt.Errorf("invalid count of x-enum-varnames. expected %d, got %d", len(field.enums), len(varNames))
		}

		field.enumVarNames = nil

		for _, v := range varNames {
			field.enumVarNames = append(field.enumVarNames, v)
		}

		if field.schemaType == ARRAY {
			// Add the var names in the items schema
			if schema.Items.Schema.Extensions == nil {
				schema.Items.Schema.Extensions = map[string]interface{}{}
			}
			schema.Items.Schema.Extensions[enumVarNamesExtension] = field.enumVarNames
		} else {
			// Add to top level schema
			if schema.Extensions == nil {
				schema.Extensions = map[string]interface{}{}
			}
			schema.Extensions[enumVarNamesExtension] = field.enumVarNames
		}
	}

	eleSchema := schema

	if field.schemaType == ARRAY {
		// For Array only
		schema.MaxItems = field.maxItems
		schema.MinItems = field.minItems
		schema.UniqueItems = field.unique

		if schema.Items != nil {
			eleSchema = schema.Items.Schema
		}

		eleSchema.Format = field.formatType
	}

	eleSchema.Maximum = field.maximum
	eleSchema.Minimum = field.minimum
	eleSchema.MultipleOf = field.multipleOf
	eleSchema.MaxLength = field.maxLength
	eleSchema.MinLength = field.minLength
	eleSchema.Enum = field.enums

	return nil
}

// parseEnumTags parses enum tags and populates field.enums.
func (ps *TagBaseFieldParser) parseEnumTags(enumTag string, field *structField) error {
	enumType := field.schemaType
	if field.schemaType == ARRAY {
		enumType = field.arrayType
	}

	field.enums = nil

	for _, e := range strings.Split(enumTag, ",") {
		value, err := ps.schemaHelper.DefineType(enumType, e)
		if err != nil {
			return err
		}

		field.enums = append(field.enums, value)
	}

	return nil
}
