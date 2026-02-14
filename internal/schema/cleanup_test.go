package schema

import (
	"testing"

	"github.com/go-openapi/spec"
)

func TestRemoveUnusedDefinitions(t *testing.T) {
	t.Run("removes unused definitions", func(t *testing.T) {
		// Arrange
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Definitions: map[string]spec.Schema{
					"User": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
							Properties: map[string]spec.Schema{
								"name": {
									SchemaProps: spec.SchemaProps{
										Type: []string{"string"},
									},
								},
							},
						},
					},
					"UnusedModel": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					},
				},
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/users": {
							PathItemProps: spec.PathItemProps{
								Get: &spec.Operation{
									OperationProps: spec.OperationProps{
										Responses: &spec.Responses{
											ResponsesProps: spec.ResponsesProps{
												StatusCodeResponses: map[int]spec.Response{
													200: {
														ResponseProps: spec.ResponseProps{
															Schema: RefSchema("User"),
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Act
		RemoveUnusedDefinitions(swagger)

		// Assert
		if len(swagger.Definitions) != 1 {
			t.Errorf("expected 1 definition, got %d", len(swagger.Definitions))
		}
		if _, ok := swagger.Definitions["User"]; !ok {
			t.Error("expected User definition to remain")
		}
		if _, ok := swagger.Definitions["UnusedModel"]; ok {
			t.Error("expected UnusedModel to be removed")
		}
	})

	t.Run("keeps transitively referenced definitions", func(t *testing.T) {
		// Arrange
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Definitions: map[string]spec.Schema{
					"Address": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					},
					"User": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
							Properties: map[string]spec.Schema{
								"address": *RefSchema("Address"),
							},
						},
					},
					"UnusedModel": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					},
				},
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/users": {
							PathItemProps: spec.PathItemProps{
								Get: &spec.Operation{
									OperationProps: spec.OperationProps{
										Responses: &spec.Responses{
											ResponsesProps: spec.ResponsesProps{
												StatusCodeResponses: map[int]spec.Response{
													200: {
														ResponseProps: spec.ResponseProps{
															Schema: RefSchema("User"),
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Act
		RemoveUnusedDefinitions(swagger)

		// Assert
		if len(swagger.Definitions) != 2 {
			t.Errorf("expected 2 definitions, got %d", len(swagger.Definitions))
		}
		if _, ok := swagger.Definitions["User"]; !ok {
			t.Error("expected User definition to remain")
		}
		if _, ok := swagger.Definitions["Address"]; !ok {
			t.Error("expected Address definition to remain (transitively referenced)")
		}
		if _, ok := swagger.Definitions["UnusedModel"]; ok {
			t.Error("expected UnusedModel to be removed")
		}
	})

	t.Run("handles nil swagger", func(t *testing.T) {
		// Act & Assert - should not panic
		RemoveUnusedDefinitions(nil)
	})

	t.Run("handles nil definitions", func(t *testing.T) {
		// Arrange
		swagger := &spec.Swagger{}

		// Act & Assert - should not panic
		RemoveUnusedDefinitions(swagger)
	})

	t.Run("handles array item references", func(t *testing.T) {
		// Arrange
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Definitions: map[string]spec.Schema{
					"User": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					},
					"UnusedModel": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					},
				},
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/users": {
							PathItemProps: spec.PathItemProps{
								Get: &spec.Operation{
									OperationProps: spec.OperationProps{
										Responses: &spec.Responses{
											ResponsesProps: spec.ResponsesProps{
												StatusCodeResponses: map[int]spec.Response{
													200: {
														ResponseProps: spec.ResponseProps{
															Schema: &spec.Schema{
																SchemaProps: spec.SchemaProps{
																	Type: []string{"array"},
																	Items: &spec.SchemaOrArray{
																		Schema: RefSchema("User"),
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Act
		RemoveUnusedDefinitions(swagger)

		// Assert
		if len(swagger.Definitions) != 1 {
			t.Errorf("expected 1 definition, got %d", len(swagger.Definitions))
		}
		if _, ok := swagger.Definitions["User"]; !ok {
			t.Error("expected User definition to remain")
		}
	})

	t.Run("handles parameter references", func(t *testing.T) {
		// Arrange
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Definitions: map[string]spec.Schema{
					"CreateUserRequest": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					},
					"UnusedModel": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					},
				},
				Paths: &spec.Paths{
					Paths: map[string]spec.PathItem{
						"/users": {
							PathItemProps: spec.PathItemProps{
								Post: &spec.Operation{
									OperationProps: spec.OperationProps{
										Parameters: []spec.Parameter{
											{
												ParamProps: spec.ParamProps{
													Schema: RefSchema("CreateUserRequest"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Act
		RemoveUnusedDefinitions(swagger)

		// Assert
		if len(swagger.Definitions) != 1 {
			t.Errorf("expected 1 definition, got %d", len(swagger.Definitions))
		}
		if _, ok := swagger.Definitions["CreateUserRequest"]; !ok {
			t.Error("expected CreateUserRequest definition to remain")
		}
	})
}

func TestCollectRefs(t *testing.T) {
	t.Run("collects refs from schema", func(t *testing.T) {
		// Arrange
		used := make(map[string]bool)
		schema := RefSchema("User")

		// Act
		collectRefs(schema, used)

		// Assert
		if !used["User"] {
			t.Error("expected User to be marked as used")
		}
	})

	t.Run("collects refs from properties", func(t *testing.T) {
		// Arrange
		used := make(map[string]bool)
		schema := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"user": *RefSchema("User"),
					"post": *RefSchema("Post"),
				},
			},
		}

		// Act
		collectRefs(schema, used)

		// Assert
		if !used["User"] {
			t.Error("expected User to be marked as used")
		}
		if !used["Post"] {
			t.Error("expected Post to be marked as used")
		}
	})
}

func TestCollectSchemaRefs(t *testing.T) {
	t.Run("handles nil schema", func(t *testing.T) {
		// Arrange
		used := make(map[string]bool)

		// Act & Assert - should not panic
		collectSchemaRefs(nil, used)
	})

	t.Run("collects direct ref", func(t *testing.T) {
		// Arrange
		used := make(map[string]bool)
		schema := RefSchema("User")

		// Act
		collectSchemaRefs(schema, used)

		// Assert
		if !used["User"] {
			t.Error("expected User to be marked as used")
		}
	})

	t.Run("collects refs from allOf", func(t *testing.T) {
		// Arrange
		used := make(map[string]bool)
		schema := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				AllOf: []spec.Schema{
					*RefSchema("Base"),
					*RefSchema("Extended"),
				},
			},
		}

		// Act
		collectSchemaRefs(schema, used)

		// Assert
		if !used["Base"] {
			t.Error("expected Base to be marked as used")
		}
		if !used["Extended"] {
			t.Error("expected Extended to be marked as used")
		}
	})
}
