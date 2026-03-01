// Package domain contains domain models for route parsing.
package domain

// Route represents a parsed HTTP route with all its metadata
type Route struct {
	// HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string

	// URL path (e.g., "/users/{id}")
	Path string

	// Summary is a short description of the route
	Summary string

	// Description is a longer explanation
	Description string

	// Tags for grouping routes
	Tags []string

	// Parameters for the route
	Parameters []Parameter

	// Responses keyed by status code
	Responses map[int]Response

	// Security requirements
	Security []map[string][]string

	// Consumes content types
	Consumes []string

	// Produces content types
	Produces []string

	// IsPublic indicates if the route has @public annotation
	IsPublic bool

	// Deprecated indicates if the route is deprecated
	Deprecated bool

	// OperationID is a unique identifier for the operation
	OperationID string

	// FilePath where this route was defined
	FilePath string

	// FunctionName that implements this route
	FunctionName string

	// LineNumber where the route is defined
	LineNumber int
}

// Parameter represents a route parameter
type Parameter struct {
	// Name of the parameter
	Name string

	// In specifies where the parameter is located (path, query, header, body, formData)
	In string

	// Type of the parameter (string, integer, boolean, array, object)
	Type string

	// Required indicates if the parameter is mandatory
	Required bool

	// Description of the parameter
	Description string

	// Schema for complex types
	Schema *Schema

	// Items for array types
	Items *Items

	// Default value
	Default interface{}

	// Format (e.g., "int32", "date-time")
	Format string

	// Enum values
	Enum []interface{}

	// Minimum value (for numbers)
	Minimum *float64

	// Maximum value (for numbers)
	Maximum *float64

	// MinLength (for strings)
	MinLength *float64

	// MaxLength (for strings)
	MaxLength *float64
}

// Items describes the items in an array parameter
type Items struct {
	// Type of array items
	Type string

	// Format of array items
	Format string

	// Enum values for array items
	Enum []interface{}
}

// Response represents an HTTP response
type Response struct {
	// Description of the response
	Description string

	// Schema of the response body
	Schema *Schema

	// Headers in the response
	Headers map[string]Header
}

// Header represents a response header
type Header struct {
	// Type of the header value
	Type string

	// Description of the header
	Description string

	// Format of the header value
	Format string
}

// Schema represents a data schema
type Schema struct {
	// Type of the schema (object, array, string, etc.)
	Type string

	// Ref is a reference to another schema ($ref)
	Ref string

	// Items for array schemas
	Items *Schema

	// Properties for object schemas
	Properties map[string]*Schema

	// AdditionalProperties for map types (map[string]T)
	AdditionalProperties *Schema

	// AllOf for composed schemas
	AllOf []*Schema

	// Required property names
	Required []string

	// Description of the schema
	Description string
}
