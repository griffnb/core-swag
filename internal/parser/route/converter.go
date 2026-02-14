package route

import (
	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/parser/route/domain"
)

// RouteToSpecOperation converts a domain.Route to a spec.Operation
func RouteToSpecOperation(route *domain.Route) *spec.Operation {
	if route == nil {
		return nil
	}

	operation := &spec.Operation{
		OperationProps: spec.OperationProps{
			ID:          route.OperationID,
			Summary:     route.Summary,
			Description: route.Description,
			Tags:        route.Tags,
			Consumes:    route.Consumes,
			Produces:    route.Produces,
			Deprecated:  route.Deprecated,
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: make(spec.Extensions),
		},
	}

	// Note: Source location extensions (x-path, x-function, x-line) removed
	// to match expected test output. Can be re-enabled if needed.

	// Convert parameters
	for _, param := range route.Parameters {
		operation.Parameters = append(operation.Parameters, ParameterToSpec(param))
	}

	// Convert responses
	responses := &spec.Responses{
		VendorExtensible: spec.VendorExtensible{
			Extensions: make(spec.Extensions),
		},
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: make(map[int]spec.Response),
		},
	}

	for code, resp := range route.Responses {
		responses.StatusCodeResponses[code] = ResponseToSpec(resp)
	}

	operation.Responses = responses

	// Convert security
	if len(route.Security) > 0 {
		operation.Security = route.Security
	}

	return operation
}

// ParameterToSpec converts a domain.Parameter to spec.Parameter
func ParameterToSpec(param domain.Parameter) spec.Parameter {
	specParam := spec.Parameter{
		ParamProps: spec.ParamProps{
			Name:        param.Name,
			In:          param.In,
			Required:    param.Required,
			Description: param.Description,
		},
	}

	// Handle schema for body parameters
	if param.Schema != nil {
		specParam.Schema = SchemaToSpec(param.Schema)
	} else {
		// For non-body parameters, set type directly
		specParam.Type = param.Type
		specParam.Format = param.Format

		if param.Items != nil {
			specParam.Items = &spec.Items{
				SimpleSchema: spec.SimpleSchema{
					Type:   param.Items.Type,
					Format: param.Items.Format,
				},
			}
			if len(param.Items.Enum) > 0 {
				specParam.Items.Enum = param.Items.Enum
			}
		}

		if param.Default != nil {
			specParam.Default = param.Default
		}

		if len(param.Enum) > 0 {
			specParam.Enum = param.Enum
		}

		if param.Minimum != nil {
			specParam.Minimum = param.Minimum
		}

		if param.Maximum != nil {
			specParam.Maximum = param.Maximum
		}

		if param.MinLength != nil {
			val := int64(*param.MinLength)
			specParam.MinLength = &val
		}

		if param.MaxLength != nil {
			val := int64(*param.MaxLength)
			specParam.MaxLength = &val
		}
	}

	return specParam
}

// ResponseToSpec converts a domain.Response to spec.Response
func ResponseToSpec(resp domain.Response) spec.Response {
	specResp := spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: resp.Description,
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: make(spec.Extensions),
		},
	}

	// Convert schema
	if resp.Schema != nil {
		specResp.Schema = SchemaToSpec(resp.Schema)
	}

	// Convert headers
	if len(resp.Headers) > 0 {
		specResp.Headers = make(map[string]spec.Header)
		for name, header := range resp.Headers {
			specResp.Headers[name] = HeaderToSpec(header)
		}
	}

	return specResp
}

// HeaderToSpec converts a domain.Header to spec.Header
func HeaderToSpec(header domain.Header) spec.Header {
	return spec.Header{
		SimpleSchema: spec.SimpleSchema{
			Type:   header.Type,
			Format: header.Format,
		},
		HeaderProps: spec.HeaderProps{
			Description: header.Description,
		},
	}
}

// SchemaToSpec converts a domain.Schema to spec.Schema
func SchemaToSpec(schema *domain.Schema) *spec.Schema {
	if schema == nil {
		return nil
	}

	specSchema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:        []string{},
			Description: schema.Description,
		},
	}

	// Set type if present
	if schema.Type != "" {
		specSchema.Type = []string{schema.Type}
	}

	// Handle reference
	if schema.Ref != "" {
		specSchema.Ref = spec.MustCreateRef(schema.Ref)
	}

	// Handle items for arrays
	if schema.Items != nil {
		specSchema.Items = &spec.SchemaOrArray{
			Schema: SchemaToSpec(schema.Items),
		}
	}

	// Handle properties for objects
	if len(schema.Properties) > 0 {
		specSchema.Properties = make(map[string]spec.Schema)
		for name, prop := range schema.Properties {
			if convertedSchema := SchemaToSpec(prop); convertedSchema != nil {
				specSchema.Properties[name] = *convertedSchema
			}
		}
	}

	// Handle required fields
	if len(schema.Required) > 0 {
		specSchema.Required = schema.Required
	}

	return specSchema
}
