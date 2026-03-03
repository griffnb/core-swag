// Package orchestrator coordinates all services to generate OpenAPI documentation.
package orchestrator

import (
	"fmt"
	"go/ast"
	"go/token"
	"runtime"
	"strings"
	"sync"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/domain"
	"github.com/griffnb/core-swag/internal/model"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/packages"
)

// structRefWork holds the resolved inputs for a struct schema build.
type structRefWork struct {
	baseName      string
	pkgPath       string
	typeName      string
	goPackageName string
}

// structRefResult holds the output of a single concurrent BuildAllSchemas call.
type structRefResult struct {
	schemas map[string]*spec.Schema
	base    string
}

// buildDemandDrivenSchemas builds OpenAPI schemas only for types that are
// actually referenced by route annotations. Struct types are built concurrently
// via BuildAllSchemas (which is internally thread-safe). Non-struct types
// (enums, aliases) are built sequentially via the SchemaBuilder.
func (s *Service) buildDemandDrivenSchemas(referencedTypes map[string]RefInfo) error {
	if s.swagger.Definitions == nil {
		s.swagger.Definitions = make(spec.Definitions)
	}

	// Phase 1: Resolve refs and partition into struct vs non-struct work.
	// Registry lookups are read-only so this is safe to do sequentially.
	processed := make(map[string]bool)
	var structWork []structRefWork
	var nonStructRefs []resolvedNonStructRef

	for refName, info := range referencedTypes {
		baseName, typeDef := s.resolveRef(refName, info, processed)
		if typeDef == nil {
			continue
		}

		if _, ok := typeDef.TypeSpec.Type.(*ast.StructType); ok {
			var goPackageName string
			if typeDef.File != nil && typeDef.File.Name != nil {
				goPackageName = typeDef.File.Name.Name
			}
			structWork = append(structWork, structRefWork{
				baseName:      baseName,
				pkgPath:       typeDef.PkgPath,
				typeName:      typeDef.Name(),
				goPackageName: goPackageName,
			})
		} else {
			nonStructRefs = append(nonStructRefs, resolvedNonStructRef{
				baseName: baseName,
				typeDef:  typeDef,
			})
		}
	}

	// Phase 1.5: Pre-warm packages with Syntax in a single batched call.
	// This replaces N sequential `go list` subprocesses with one batched call.
	if err := preWarmPackages(structWork, s.config.Debug); err != nil {
		// Non-fatal: concurrent builds fall back to individual loads
		// (deduplicated by singleflight).
		if s.config.Debug != nil {
			s.config.Debug.Printf("Orchestrator: preWarmPackages failed (non-fatal): %v", err)
		}
	}

	// Phase 2: Build struct schemas concurrently.
	// BuildAllSchemas creates a fresh CoreStructParser per call and only
	// touches mutex-protected global caches, so it is safe to parallelize.
	results, err := s.buildStructSchemasConcurrent(structWork)
	if err != nil {
		return err
	}

	// Phase 3: Merge struct results into swagger definitions sequentially.
	// When multiple concurrent builds produce schemas for the same name
	// (e.g., through recursive nested type resolution), prefer the schema
	// with more properties — empty schemas from failed package resolution
	// should not shadow proper schemas from the type's own build.
	for _, r := range results {
		for name, schema := range r.schemas {
			existing, exists := s.swagger.Definitions[name]
			if !exists {
				s.swagger.Definitions[name] = *schema
			} else if len(existing.Properties) == 0 && len(schema.Properties) > 0 {
				s.swagger.Definitions[name] = *schema
			}
		}
	}

	// Phase 4: Build non-struct schemas sequentially.
	// SchemaBuilder has no internal synchronization so these cannot be parallelized.
	for _, ref := range nonStructRefs {
		s.buildNonStructSchema(ref)
	}

	// Phase 5: Sync any non-struct schemas built via SchemaBuilder.
	for name, schema := range s.schemaBuilder.Definitions() {
		if _, exists := s.swagger.Definitions[name]; !exists {
			s.swagger.Definitions[name] = schema
		}
	}

	return nil
}

// buildStructSchemasConcurrent runs BuildAllSchemas for each struct type in
// parallel, bounded by NumCPU. Results are collected under a mutex and returned.
func (s *Service) buildStructSchemasConcurrent(work []structRefWork) ([]structRefResult, error) {
	var (
		mu      sync.Mutex
		results []structRefResult
	)

	sharedCache := model.NewSharedTypeCache()

	var g errgroup.Group
	g.SetLimit(runtime.NumCPU())
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: BuildAllSchemas With %d workers", runtime.NumCPU())
	}

	for _, w := range work {
		w := w
		g.Go(func() error {
			schemas, err := model.BuildAllSchemasWithCache("", w.pkgPath, w.typeName, sharedCache, w.goPackageName)
			if err != nil {
				if s.config.Debug != nil {
					s.config.Debug.Printf("Orchestrator: BuildAllSchemas FAILED for %s (pkg=%s): %v",
						w.baseName, w.pkgPath, err)
				}
				// Non-fatal: skip this type.
				return nil
			}

			if s.config.Debug != nil {
				s.config.Debug.Printf("Orchestrator: BuildAllSchemas OK for %s → %d schemas",
					w.baseName, len(schemas))
			}

			mu.Lock()
			results = append(results, structRefResult{schemas: schemas, base: w.baseName})
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("concurrent schema build: %w", err)
	}

	return results, nil
}

// resolvedNonStructRef holds a resolved non-struct type ready for sequential building.
type resolvedNonStructRef struct {
	baseName string
	typeDef  *domain.TypeSpecDef
}

// resolveRef looks up a $ref in the registry and marks it as processed.
// Returns the base name and type definition, or nil if already processed or not found.
func (s *Service) resolveRef(refName string, info RefInfo, processed map[string]bool) (string, *domain.TypeSpecDef) {
	if processed[refName] {
		return "", nil
	}

	baseName := refName
	if strings.HasSuffix(refName, "Public") {
		baseName = strings.TrimSuffix(refName, "Public")
	}

	if processed[baseName] {
		return "", nil
	}

	var typeDef *domain.TypeSpecDef
	if info.TypePath != "" {
		lookupPath := info.TypePath
		if strings.HasSuffix(lookupPath, "Public") {
			lookupPath = strings.TrimSuffix(lookupPath, "Public")
		}
		typeDef = s.registry.FindTypeSpecByFullPath(lookupPath)
	}

	if typeDef == nil {
		typeDef = s.registry.FindTypeSpecByName(baseName)
	}

	if typeDef == nil {
		if s.config.Debug != nil {
			if info.Source != "" {
				s.config.Debug.Printf("Orchestrator: Skipping unknown ref %s (not in registry) referenced by %s", refName, info.Source)
			} else {
				s.config.Debug.Printf("Orchestrator: Skipping unknown ref %s (not in registry)", refName)
			}
		}
		return "", nil
	}

	processed[baseName] = true
	return baseName, typeDef
}

// buildNonStructSchema builds a schema for a non-struct type (enum, type alias)
// using the SchemaBuilder.
func (s *Service) buildNonStructSchema(ref resolvedNonStructRef) {
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Building non-struct schema for %s (type=%T)",
			ref.baseName, ref.typeDef.TypeSpec.Type)
	}
	schemaName, err := s.schemaBuilder.BuildSchema(ref.typeDef)
	if err != nil {
		if s.config.Debug != nil {
			s.config.Debug.Printf("Orchestrator: BuildSchema FAILED for %s: %v", ref.baseName, err)
		}
	} else if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: BuildSchema OK for %s → %s", ref.baseName, schemaName)
	}
}

// preWarmPackages loads all unique package paths from the work slice in a single
// batched packages.Load call. This triggers one `go list` invocation that
// resolves everything, dramatically faster than N individual calls.
func preWarmPackages(work []structRefWork, debug Debugger) error {
	// Collect unique pkgPaths that aren't already cached with Syntax.
	seen := make(map[string]bool, len(work))
	var paths []string
	for _, w := range work {
		if seen[w.pkgPath] {
			continue
		}
		seen[w.pkgPath] = true
		if model.IsPackageCachedWithSyntax(w.pkgPath) {
			continue
		}
		paths = append(paths, w.pkgPath)
	}

	if len(paths) == 0 {
		if debug != nil {
			debug.Printf("Orchestrator: preWarmPackages: all %d packages already cached with syntax", len(work))
		}
		return nil
	}

	if debug != nil {
		debug.Printf("Orchestrator: preWarmPackages: loading %d packages in single batch", len(paths))
	}

	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedName | packages.NeedImports | packages.NeedDeps,
		Fset: token.NewFileSet(),
	}
	pkgs, err := packages.Load(cfg, paths...)
	if err != nil {
		return fmt.Errorf("batched packages.Load: %w", err)
	}

	model.SeedGlobalPackageCache(pkgs)
	model.SeedEnumPackageCache(pkgs)

	if debug != nil {
		debug.Printf("Orchestrator: preWarmPackages: seeded %d packages", len(pkgs))
	}
	return nil
}
