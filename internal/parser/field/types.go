package field

// Schema type constants
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
)

// Naming strategy constants
const (
	// CamelCase indicates using CamelCase strategy for struct field.
	CamelCase = "camelcase"
	// PascalCase indicates using PascalCase strategy for struct field.
	PascalCase = "pascalcase"
	// SnakeCase indicates using SnakeCase strategy for struct field.
	SnakeCase = "snakecase"
)

// Tag names
const (
	requiredLabel    = "required"
	optionalLabel    = "optional"
	omitEmptyLabel   = "omitempty"
	swaggerTypeTag   = "swaggertype"
	swaggerIgnoreTag = "swaggerignore"
	jsonTag          = "json"
	formTag          = "form"
	headerTag        = "header"
	bindingTag       = "binding"
	validateTag      = "validate"
	uriTag           = "uri"
	formatTag        = "format"
	titleTag         = "title"
	enumsTag         = "enums"
	maximumTag       = "maximum"
	minimumTag       = "minimum"
	defaultTag       = "swag_default"
	exampleTag       = "example"
	minLengthTag     = "minLength"
	maxLengthTag     = "maxLength"
	readOnlyTag      = "readonly"
	multipleOfTag    = "multipleOf"
	extensionsTag    = "extensions"
)

// Extension constants
const (
	enumVarNamesExtension = "x-enum-varnames"
)

// UTF-8 special character codes
const (
	utf8HexComma = "0x2C"
	utf8Pipe     = "0x7C"
)
