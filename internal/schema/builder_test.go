package schema

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/domain"
)

func TestNewBuilder(t *testing.T) {
	t.Run("creates new builder with empty maps", func(t *testing.T) {
		// Act
		builder := NewBuilder()

		// Assert
		if builder == nil {
			t.Fatal("expected builder to not be nil")
		}
		if builder.definitions == nil {
			t.Error("expected definitions map to be initialized")
		}
		if builder.parsedSchemas == nil {
			t.Error("expected parsedSchemas map to be initialized")
		}
	})
}

func TestBuilderService_AddDefinition(t *testing.T) {
	t.Run("adds definition successfully", func(t *testing.T) {
		// Arrange
		builder := NewBuilder()
		schema := spec.Schema{
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
		}

		// Act
		err := builder.AddDefinition("User", schema)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		defs := builder.Definitions()
		if len(defs) != 1 {
			t.Errorf("expected 1 definition, got %d", len(defs))
		}
		if _, ok := defs["User"]; !ok {
			t.Error("expected User definition to exist")
		}
	})

	t.Run("overwrites existing definition", func(t *testing.T) {
		// Arrange
		builder := NewBuilder()
		schema1 := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
			},
		}
		schema2 := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:        []string{"object"},
				Description: "Updated",
			},
		}

		// Act
		_ = builder.AddDefinition("User", schema1)
		err := builder.AddDefinition("User", schema2)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		defs := builder.Definitions()
		if defs["User"].Description != "Updated" {
			t.Error("expected definition to be updated")
		}
	})
}

func TestBuilderService_GetDefinition(t *testing.T) {
	t.Run("retrieves existing definition", func(t *testing.T) {
		// Arrange
		builder := NewBuilder()
		schema := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
			},
		}
		_ = builder.AddDefinition("User", schema)

		// Act
		retrieved, found := builder.GetDefinition("User")

		// Assert
		if !found {
			t.Error("expected definition to be found")
		}
		if len(retrieved.Type) != 1 || retrieved.Type[0] != "object" {
			t.Error("expected retrieved schema to match")
		}
	})

	t.Run("returns false for non-existent definition", func(t *testing.T) {
		// Arrange
		builder := NewBuilder()

		// Act
		_, found := builder.GetDefinition("NonExistent")

		// Assert
		if found {
			t.Error("expected definition to not be found")
		}
	})
}

func TestBuilderService_BuildSchema(t *testing.T) {
	t.Run("builds schema for simple struct", func(t *testing.T) {
		// Arrange
		builder := NewBuilder()
		src := `package test
type User struct {
	Name string
	Age  int
}`
		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}

		// Find the User type spec
		var typeSpec *ast.TypeSpec
		for _, decl := range astFile.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "User" {
						typeSpec = ts
						break
					}
				}
			}
		}

		if typeSpec == nil {
			t.Fatal("could not find User type spec")
		}

		typeDef := &domain.TypeSpecDef{
			File:       astFile,
			TypeSpec:   typeSpec,
			PkgPath:    "github.com/test/pkg",
			ParentSpec: nil,
		}
		typeDef.SetSchemaName()

		// Act
		schemaName, err := builder.BuildSchema(typeDef)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if schemaName == "" {
			t.Error("expected schema name to not be empty")
		}
		defs := builder.Definitions()
		if len(defs) == 0 {
			t.Error("expected at least one definition to be created")
		}
	})

	t.Run("returns cached schema name for already parsed type", func(t *testing.T) {
		// Arrange
		builder := NewBuilder()
		src := `package test
type User struct {
	Name string
}`
		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}

		var typeSpec *ast.TypeSpec
		for _, decl := range astFile.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "User" {
						typeSpec = ts
						break
					}
				}
			}
		}

		typeDef := &domain.TypeSpecDef{
			File:       astFile,
			TypeSpec:   typeSpec,
			PkgPath:    "github.com/test/pkg",
			ParentSpec: nil,
		}
		typeDef.SetSchemaName()

		// Act - build twice
		name1, _ := builder.BuildSchema(typeDef)
		name2, _ := builder.BuildSchema(typeDef)

		// Assert
		if name1 != name2 {
			t.Errorf("expected same schema name, got %s and %s", name1, name2)
		}
		defs := builder.Definitions()
		if len(defs) != 1 {
			t.Errorf("expected 1 definition for duplicate build, got %d", len(defs))
		}
	})
}

func TestBuilderService_Definitions(t *testing.T) {
	t.Run("returns all definitions", func(t *testing.T) {
		// Arrange
		builder := NewBuilder()
		schema1 := spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"object"}}}
		schema2 := spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{"string"}}}
		_ = builder.AddDefinition("User", schema1)
		_ = builder.AddDefinition("Product", schema2)

		// Act
		defs := builder.Definitions()

		// Assert
		if len(defs) != 2 {
			t.Errorf("expected 2 definitions, got %d", len(defs))
		}
		if _, ok := defs["User"]; !ok {
			t.Error("expected User definition")
		}
		if _, ok := defs["Product"]; !ok {
			t.Error("expected Product definition")
		}
	})
}
