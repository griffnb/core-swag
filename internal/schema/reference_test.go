package schema

import (
	"testing"

	"github.com/go-openapi/spec"
)

func TestRefSchema(t *testing.T) {
	t.Run("creates reference schema", func(t *testing.T) {
		// Act
		schema := RefSchema("User")

		// Assert
		if schema == nil {
			t.Fatal("expected schema to not be nil")
		}
		if schema.Ref.String() != "#/definitions/User" {
			t.Errorf("expected ref to be #/definitions/User, got %s", schema.Ref.String())
		}
	})

	t.Run("handles complex type names", func(t *testing.T) {
		// Act
		schema := RefSchema("api.v1.User")

		// Assert
		if schema.Ref.String() != "#/definitions/api.v1.User" {
			t.Errorf("expected ref to be #/definitions/api.v1.User, got %s", schema.Ref.String())
		}
	})
}

func TestIsRefSchema(t *testing.T) {
	t.Run("returns true for reference schema", func(t *testing.T) {
		// Arrange
		schema := RefSchema("User")

		// Act
		isRef := IsRefSchema(schema)

		// Assert
		if !isRef {
			t.Error("expected schema to be a reference schema")
		}
	})

	t.Run("returns false for non-reference schema", func(t *testing.T) {
		// Arrange
		schema := &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
			},
		}

		// Act
		isRef := IsRefSchema(schema)

		// Assert
		if isRef {
			t.Error("expected schema to not be a reference schema")
		}
	})

	t.Run("returns false for nil schema", func(t *testing.T) {
		// Act
		isRef := IsRefSchema(nil)

		// Assert
		if isRef {
			t.Error("expected nil schema to not be a reference schema")
		}
	})
}

func TestResolveReferences(t *testing.T) {
	t.Run("resolves references in schema properties", func(t *testing.T) {
		// Arrange
		definitions := map[string]spec.Schema{
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
			"Post": {
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					Properties: map[string]spec.Schema{
						"author": *RefSchema("User"),
					},
				},
			},
		}

		// Act
		err := ResolveReferences(definitions)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		// The Post.author should still be a reference (ResolveReferences validates, doesn't inline)
		post := definitions["Post"]
		author := post.Properties["author"]
		if author.Ref.String() != "#/definitions/User" {
			t.Error("expected author to still reference User")
		}
	})

	t.Run("handles array item references", func(t *testing.T) {
		// Arrange
		definitions := map[string]spec.Schema{
			"User": {
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
				},
			},
			"UserList": {
				SchemaProps: spec.SchemaProps{
					Type: []string{"array"},
					Items: &spec.SchemaOrArray{
						Schema: RefSchema("User"),
					},
				},
			},
		}

		// Act
		err := ResolveReferences(definitions)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("handles nested references", func(t *testing.T) {
		// Arrange
		definitions := map[string]spec.Schema{
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
			"Post": {
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					Properties: map[string]spec.Schema{
						"author": *RefSchema("User"),
					},
				},
			},
		}

		// Act
		err := ResolveReferences(definitions)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestGetRefName(t *testing.T) {
	t.Run("extracts name from valid reference", func(t *testing.T) {
		// Act
		name := getRefName("#/definitions/User")

		// Assert
		if name != "User" {
			t.Errorf("expected 'User', got '%s'", name)
		}
	})

	t.Run("extracts complex name from reference", func(t *testing.T) {
		// Act
		name := getRefName("#/definitions/api.v1.User")

		// Assert
		if name != "api.v1.User" {
			t.Errorf("expected 'api.v1.User', got '%s'", name)
		}
	})

	t.Run("returns empty for invalid reference", func(t *testing.T) {
		// Act
		name := getRefName("invalid-ref")

		// Assert
		if name != "" {
			t.Errorf("expected empty string, got '%s'", name)
		}
	})

	t.Run("returns empty for wrong prefix", func(t *testing.T) {
		// Act
		name := getRefName("#/components/User")

		// Assert
		if name != "" {
			t.Errorf("expected empty string, got '%s'", name)
		}
	})
}
