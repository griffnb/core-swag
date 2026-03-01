// Package orchestrator coordinates all services to generate OpenAPI documentation.
package orchestrator

import (
	"go/ast"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/domain"
	"github.com/griffnb/core-swag/internal/model"
)

// buildDemandDrivenSchemas builds OpenAPI schemas only for types that are
// actually referenced by route annotations. For struct types it uses
// BuildAllSchemas which generates both public and non-public variants and
// resolves transitive nested dependencies. For non-struct types (enums, aliases)
// it uses the SchemaBuilder. After building, it adds redirect definitions so
// that unqualified $ref names (used by route annotations) resolve to the
// qualified definition names.
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

	// Add unqualified-name redirects. Route annotations often reference types
	// without package prefixes (e.g., "AdminForceClassifyInput" instead of
	// "account_classifications.AdminForceClassifyInput"). Create $ref redirects
	// from unqualified names to the qualified definition, but only if the
	// unqualified name doesn't already exist as a definition.
	s.addUnqualifiedRedirects()

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

	// Try to find the type in the registry
	typeDef := s.registry.FindTypeSpecByName(baseName)
	if typeDef == nil {
		// Route $refs often omit the package prefix. Search by unqualified name.
		typeDef = s.findUnqualifiedType(baseName)
	}
	if typeDef == nil {
		// For NotUnique types, the registry key is full-path format. Try
		// matching by Go package name + type name (e.g., "tag.Tag" matches
		// a typeDef where File.Name.Name == "tag" and Name() == "Tag").
		typeDef = s.findTypeByShortName(baseName)
	}

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

// findUnqualifiedType searches the registry for a type by its unqualified
// name (without package prefix). This handles the case where route annotations
// reference types defined in the same controller file without a package prefix.
func (s *Service) findUnqualifiedType(typeName string) *domain.TypeSpecDef {
	// Only search for truly unqualified names (no dot)
	if strings.Contains(typeName, ".") {
		return nil
	}

	for _, typeDef := range s.registry.UniqueDefinitions() {
		if typeDef == nil {
			continue
		}
		if typeDef.Name() == typeName {
			return typeDef
		}
	}
	return nil
}

// findTypeByShortName searches the registry for a NotUnique type that matches
// a short qualified name like "tag.Tag". When types are marked NotUnique, their
// registry key changes to a full-path format (e.g., "github_com_..._tag.Tag"),
// making them unreachable via FindTypeSpecByName("tag.Tag"). This method
// scans all definitions to find a match by Go package name + type name.
func (s *Service) findTypeByShortName(qualifiedName string) *domain.TypeSpecDef {
	dotIdx := strings.LastIndex(qualifiedName, ".")
	if dotIdx < 0 {
		return nil
	}
	pkgName := qualifiedName[:dotIdx]
	typeName := qualifiedName[dotIdx+1:]

	for _, typeDef := range s.registry.UniqueDefinitions() {
		if typeDef == nil {
			continue
		}
		if typeDef.Name() == typeName &&
			typeDef.File != nil && typeDef.File.Name != nil &&
			typeDef.File.Name.Name == pkgName {
			return typeDef
		}
	}
	return nil
}

// addUnqualifiedRedirects creates redirect definitions from unqualified type
// names to their qualified definitions. This allows route $refs like
// "#/definitions/AdminForceClassifyInput" to resolve to
// "#/definitions/account_classifications.AdminForceClassifyInput".
func (s *Service) addUnqualifiedRedirects() {
	// Collect qualified definition names first to avoid modifying map during iteration
	redirects := make(map[string]string) // unqualified → qualified

	for defName := range s.swagger.Definitions {
		dotIdx := strings.LastIndex(defName, ".")
		if dotIdx < 0 {
			continue
		}
		unqualified := defName[dotIdx+1:]
		if _, exists := s.swagger.Definitions[unqualified]; exists {
			// Already exists as its own definition — don't redirect
			continue
		}
		if existing, alreadyRedirected := redirects[unqualified]; alreadyRedirected {
			// Ambiguous — multiple qualified names for the same unqualified name.
			// Keep the first one found (arbitrary but deterministic within a run).
			if s.config.Debug != nil {
				s.config.Debug.Printf("Orchestrator: Ambiguous redirect for %s: %s vs %s", unqualified, existing, defName)
			}
			continue
		}
		redirects[unqualified] = defName
	}

	for unqualified, qualified := range redirects {
		s.swagger.Definitions[unqualified] = spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/" + qualified),
			},
		}
	}
}
