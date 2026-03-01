// Package orchestrator coordinates all services to generate OpenAPI documentation.
package orchestrator

import (
	"go/ast"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/model"
)

// buildDemandDrivenSchemas builds OpenAPI schemas only for types that are
// actually referenced by route annotations. For struct types it uses
// BuildAllSchemas which generates both public and non-public variants and
// resolves transitive nested dependencies. For non-struct types (enums, aliases)
// it uses the SchemaBuilder.
func (s *Service) buildDemandDrivenSchemas(referencedTypes map[string]bool) error {
	if s.swagger.Definitions == nil {
		s.swagger.Definitions = make(spec.Definitions)
	}

	processed := make(map[string]bool)

	for refName := range referencedTypes {
		if err := s.buildSchemaForRef(refName, processed); err != nil {
			return err
		}
	}

	// Sync any non-struct schemas built via SchemaBuilder
	for name, schema := range s.schemaBuilder.Definitions() {
		if _, exists := s.swagger.Definitions[name]; !exists {
			s.swagger.Definitions[name] = schema
		}
	}

	return nil
}

// buildSchemaForRef resolves a $ref name and builds its schema.
func (s *Service) buildSchemaForRef(refName string, processed map[string]bool) error {
	if processed[refName] {
		return nil
	}

	// Strip Public suffix to find the base type
	baseName := refName
	if strings.HasSuffix(refName, "Public") {
		baseName = strings.TrimSuffix(refName, "Public")
	}

	if processed[baseName] {
		return nil
	}

	// Direct qualified lookup — all refs should now use package.Type format
	typeDef := s.registry.FindTypeSpecByName(baseName)

	if typeDef == nil {
		if s.config.Debug != nil {
			s.config.Debug.Printf("Orchestrator: Skipping unknown ref %s (not in registry)", refName)
		}
		return nil
	}

	processed[baseName] = true

	// For struct types, use BuildAllSchemas which handles Public variants
	// and transitive nested type dependencies.
	if _, ok := typeDef.TypeSpec.Type.(*ast.StructType); ok {
		// Pass the Go package name from the AST so definition keys match
		// route $refs. The Go package name (e.g., "stripe") may differ from
		// the pkgPath's last segment (e.g., "v84").
		var goPackageName string
		if typeDef.File != nil && typeDef.File.Name != nil {
			goPackageName = typeDef.File.Name.Name
		}
		schemas, err := model.BuildAllSchemas("", typeDef.PkgPath, typeDef.Name(), goPackageName)
		if err != nil {
			if s.config.Debug != nil {
				s.config.Debug.Printf("Orchestrator: BuildAllSchemas FAILED for %s (pkg=%s): %v",
					baseName, typeDef.PkgPath, err)
			}
			// Non-fatal: skip and continue
			return nil
		}
		for name, schema := range schemas {
			if _, exists := s.swagger.Definitions[name]; !exists {
				s.swagger.Definitions[name] = *schema
			}
			processed[name] = true
		}
		if s.config.Debug != nil {
			s.config.Debug.Printf("Orchestrator: BuildAllSchemas OK for %s → %d schemas", baseName, len(schemas))
		}
		return nil
	}

	// Non-struct type (enum, type alias) — use SchemaBuilder
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Building non-struct schema for %s (type=%T)",
			baseName, typeDef.TypeSpec.Type)
	}
	schemaName, err := s.schemaBuilder.BuildSchema(typeDef)
	if err != nil {
		if s.config.Debug != nil {
			s.config.Debug.Printf("Orchestrator: BuildSchema FAILED for %s: %v", baseName, err)
		}
	} else if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: BuildSchema OK for %s → %s", baseName, schemaName)
	}
	return nil
}

