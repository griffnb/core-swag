# Performance Optimization: Design Document

## Overview

This design restructures the orchestrator from an eager, supply-driven pipeline into a demand-driven pipeline. Instead of parsing every struct in every file and building schemas for all types, the system:

1. Parses routes first to discover which types are actually referenced
2. Builds schemas only for those types (plus their transitive dependencies)
3. Seeds downstream caches from the loader to eliminate redundant `packages.Load()` calls
4. Parallelizes route parsing across files

The result is less total work, fewer `packages.Load()` calls, and better CPU utilization.

## Architecture

### Current Flow (Eager / Backwards)

```
LoadWithGoPackages()
  └─ packages.Load() ─────────── [loads ALL packages once]
  └─ Returns LoadResult{Files, Packages}
       │
       ▼
Phase 2: for file in Files ───── registry.CollectAstFile()     [sequential]
Phase 2b: registry.ParseTypes()                                 [sequential]
       │
       ▼
Phase 3: baseParser.ParseGeneralAPIInfo()                       [single file]
       │
       ▼
Phase 3.5: for file in Files ── structParser.ParseFile()        [sequential, EVERY struct]
       │                           └─ schemaBuilder.AddDefinition() for ALL structs
       │                           └─ also builds Public variant for ALL structs
       │                           └─ triggers packages.Load() per type [REDUNDANT]
       ▼
Phase 4: for file in Files ──── routeParser.ParseRoutes()       [sequential]
       │                           └─ produces $ref strings (never reads schemas)
       ▼
Phase 5: for typeDef in ALL ─── schemaBuilder.BuildSchema()     [sequential, ALL types]
                                   └─ skips structs already built in 3.5
                                   └─ triggers packages.Load() per type [REDUNDANT]
```

### Proposed Flow (Demand-Driven)

```
LoadWithGoPackages()
  └─ packages.Load() ─────────── [loads ALL packages once]
  └─ Returns LoadResult{Files, Packages}
       │
       ▼
NEW: SeedGlobalPackageCache(Packages)    [pre-populate caches, no more redundant loads]
NEW: SeedEnumPackageCache(Packages)
       │
       ▼
Phase 2: for file in Files ───── registry.CollectAstFile()     [sequential, unchanged]
Phase 2b: registry.ParseTypes()                                 [sequential, unchanged]
       │
       ▼
Phase 3: baseParser.ParseGeneralAPIInfo()                       [single file, unchanged]
       │
       ▼
Phase 4: errgroup(Files) ────── routeParser.ParseRoutes()       [PARALLEL]
       │  └─ collect routes per goroutine
       │  └─ sort by file path, merge into swagger.Paths
       │  └─ walk all routes → collect referenced type names
       ▼
Phase 5: for type in REFERENCED ─ buildSchemaOnDemand()         [only referenced types]
                                    └─ CoreStructParser.LookupStructFields()
                                    │    └─ globalPackageCache HIT [no packages.Load!]
                                    └─ buildSchemasRecursive() for transitive deps
                                    └─ Public variants built for referenced types
```

**Phase 3.5 is eliminated.** Phase 5 only processes types found in route `$ref` strings.

## Components and Interfaces

### Component 1: Route Ref Collector

**Location:** New file `internal/orchestrator/refs.go`

A utility that walks parsed routes and extracts all referenced type names from `$ref` strings.

```go
// CollectReferencedTypes walks all routes and returns the unique set of type names
// referenced in $ref strings from parameters, responses, and nested schemas.
func CollectReferencedTypes(routes []*routedomain.Route) map[string]bool {
    refs := make(map[string]bool)
    for _, route := range routes {
        // Walk parameters
        for _, param := range route.Parameters {
            if param.Schema != nil {
                collectRefsFromSchema(param.Schema, refs)
            }
        }
        // Walk responses
        for _, resp := range route.Responses {
            if resp.Schema != nil {
                collectRefsFromSchema(resp.Schema, refs)
            }
        }
    }
    return refs
}

func collectRefsFromSchema(schema *routedomain.Schema, refs map[string]bool) {
    if schema == nil {
        return
    }
    if schema.Ref != "" {
        // Strip "#/definitions/" prefix
        typeName := strings.TrimPrefix(schema.Ref, "#/definitions/")
        if typeName != "" {
            refs[typeName] = true
        }
    }
    // Recurse into nested structures
    if schema.Items != nil {
        collectRefsFromSchema(schema.Items, refs)
    }
    for _, prop := range schema.Properties {
        collectRefsFromSchema(prop, refs)
    }
    for _, allOf := range schema.AllOf {
        collectRefsFromSchema(allOf, refs)
    }
}
```

**Design decision:** This is a simple, pure function — no state, no side effects. It runs once after route parsing completes. The `map[string]bool` return is the "demand signal" that drives Phase 5.

### Component 2: Demand-Driven Schema Building

**Location:** Modified `internal/orchestrator/service.go` Phase 5, plus modifications to `internal/schema/builder.go`

The orchestrator's Phase 5 changes from iterating `registry.UniqueDefinitions()` (ALL types) to iterating only the collected referenced type names.

**Orchestrator change:**
```go
// Phase 5: Build schemas only for route-referenced types
referencedTypes := CollectReferencedTypes(allRoutes)

for typeName := range referencedTypes {
    // Look up the type in the registry
    typeDef := s.registry.FindTypeSpecByName(typeName)
    if typeDef == nil {
        // Type from annotation not found in registry - skip with warning
        if s.config.Debug != nil {
            s.config.Debug.Printf("Warning: referenced type %s not found in registry", typeName)
        }
        continue
    }

    _, err = s.schemaBuilder.BuildSchema(typeDef)
    if err != nil {
        return nil, fmt.Errorf("failed to build schema for %s: %w", typeName, err)
    }
}
```

**Public variant handling:** The existing `buildSchemasRecursive()` in `struct_field_lookup.go` already builds both base and Public variants when it encounters nested types. For top-level route-referenced types:

- If a route references `account.AccountPublic`, the collector captures that name directly
- `BuildSchema` needs to handle the `Public` suffix — look up the base type `account.Account` in the registry, then build with Public mode
- If a route references `account.Account`, only the base variant is built (unless `account.AccountPublic` is also in the referenced set)

```go
// In the schema builder or orchestrator, when processing a Public type name:
if strings.HasSuffix(typeName, "Public") {
    baseTypeName := strings.TrimSuffix(typeName, "Public")
    typeDef := s.registry.FindTypeSpecByName(baseTypeName)
    // Build with Public mode enabled
    _, err = s.schemaBuilder.BuildSchemaPublic(typeDef)
}
```

**SchemaBuilder additions needed:**
- `BuildSchemaPublic(typeSpec)` — same as `BuildSchema` but uses Public mode in `CoreStructParser.LookupStructFields` and `BuildSpecSchema`
- Or: extend `BuildSchema` with an option/flag for Public mode

**Transitive resolution:** Already handled by `buildSchemasRecursive()` in `struct_field_lookup.go`. When building a schema, `StructField.ToSpecSchema()` returns `nestedTypes` — each nested type gets recursively built. The `processed` map prevents infinite loops. This means building `account.Account` will automatically build `address.Address` if Account has an Address field.

**Design decision:** We don't need to pre-compute the transitive closure of referenced types. `BuildSchema` + `buildSchemasRecursive` already chase dependencies. We just need to seed the process with the direct route references.

### Component 3: Registry Type Lookup by Name

**Location:** `internal/registry/service.go`

The registry currently exposes `FindTypeSpec(typeName string, file *ast.File)` which needs an AST file context. For demand-driven building, we need to look up types by their qualified name (e.g., `account.Account`).

The registry already has `uniqueDefinitions map[string]*domain.TypeSpecDef` keyed by qualified name. We need a simple accessor:

```go
// FindTypeSpecByName looks up a type definition by its qualified name (e.g., "account.Account").
// Returns nil if not found.
func (s *Service) FindTypeSpecByName(name string) *domain.TypeSpecDef {
    return s.uniqueDefinitions[name]
}
```

**Design decision:** This is a trivial addition — the data structure already exists, we just need a public accessor. No new indexing required.

### Component 4: Cache Seeding Functions

**Location:** `internal/model/struct_field_lookup.go` and `internal/model/enum_lookup.go`

Two exported functions that pre-populate the existing global caches from the loader's packages:

```go
// internal/model/struct_field_lookup.go

// SeedGlobalPackageCache pre-populates the global package cache with
// packages already loaded by the loader, avoiding redundant packages.Load() calls.
func SeedGlobalPackageCache(pkgs []*packages.Package) {
    globalCacheMutex.Lock()
    defer globalCacheMutex.Unlock()
    seedPackagesRecursive(pkgs, make(map[string]bool))
}

func seedPackagesRecursive(pkgs []*packages.Package, visited map[string]bool) {
    for _, pkg := range pkgs {
        if pkg == nil || visited[pkg.PkgPath] {
            continue
        }
        visited[pkg.PkgPath] = true
        if _, exists := globalPackageCache[pkg.PkgPath]; !exists {
            globalPackageCache[pkg.PkgPath] = pkg
        }
        for _, imp := range pkg.Imports {
            seedPackagesRecursive([]*packages.Package{imp}, visited)
        }
    }
}
```

```go
// internal/model/enum_lookup.go

// SeedEnumPackageCache pre-populates the enum package cache with
// packages already loaded by the loader.
func SeedEnumPackageCache(pkgs []*packages.Package) {
    enumCacheMutex.Lock()
    defer enumCacheMutex.Unlock()
    seedEnumPackagesRecursive(pkgs, make(map[string]bool))
}

func seedEnumPackagesRecursive(pkgs []*packages.Package, visited map[string]bool) {
    for _, pkg := range pkgs {
        if pkg == nil || visited[pkg.PkgPath] {
            continue
        }
        visited[pkg.PkgPath] = true
        if _, exists := enumPackageCache[pkg.PkgPath]; !exists {
            enumPackageCache[pkg.PkgPath] = pkg
        }
        for _, imp := range pkg.Imports {
            seedEnumPackagesRecursive([]*packages.Package{imp}, visited)
        }
    }
}
```

**Orchestrator call site** (after loading, before any schema work):
```go
if loadResult.Packages != nil {
    model.SeedGlobalPackageCache(loadResult.Packages)
    model.SeedEnumPackageCache(loadResult.Packages)
}
```

**Design decision:** Recursive seeding walks `pkg.Imports` because `CoreStructParser.LookupStructFields` loads packages by import path — if a struct field references a type from an imported package, that import path needs to be in the cache. The loader's `walkPackages` already recurses imports, but we seed unconditionally to cover all resolved transitive imports.

### Component 5: Parallel Route Parsing

**Location:** Modified `internal/orchestrator/service.go` Phase 4

Route parsing is parallelized across files. `route.Service.ParseRoutes()` has no mutable state — it's safe for concurrent use per-file.

```go
// Phase 4: Parse routes in parallel, collect results
type fileRoutes struct {
    filePath string
    routes   []*routedomain.Route
}

var (
    collected []fileRoutes
    mu        sync.Mutex
)

g, ctx := errgroup.WithContext(context.Background())
g.SetLimit(runtime.NumCPU())

for astFile, fileInfo := range loadResult.Files {
    astFile, fileInfo := astFile, fileInfo // capture loop vars
    g.Go(func() error {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        routes, err := s.routeParser.ParseRoutes(astFile, fileInfo.Path, fileInfo.FileSet)
        if err != nil {
            return fmt.Errorf("parse routes in %s: %w", fileInfo.Path, err)
        }
        if len(routes) > 0 {
            mu.Lock()
            collected = append(collected, fileRoutes{filePath: fileInfo.Path, routes: routes})
            mu.Unlock()
        }
        return nil
    })
}

if err := g.Wait(); err != nil {
    return err
}

// Sort for deterministic output
sort.Slice(collected, func(i, j int) bool {
    return collected[i].filePath < collected[j].filePath
})

// Merge into swagger.Paths sequentially + collect all routes for ref extraction
var allRoutes []*routedomain.Route
for _, fr := range collected {
    for _, r := range fr.routes {
        allRoutes = append(allRoutes, r)
        operation := route.RouteToSpecOperation(r)
        if operation == nil {
            continue
        }
        // ... existing pathItem logic (unchanged)
    }
}

// Collect referenced types for demand-driven schema building
referencedTypes := CollectReferencedTypes(allRoutes)
```

**Existing pattern:** `internal/format/format.go:63` already uses `errgroup.Group` + `SetLimit(runtime.GOMAXPROCS(0))`.

**Design decision:** Sort by file path before merging ensures deterministic `swagger.Paths` insertion order regardless of goroutine scheduling.

### Component 6: Shared FileSet

**Location:** `internal/model/struct_field_lookup.go` and `internal/model/enum_lookup.go`

For fallback `packages.Load()` calls (cache misses), reuse a shared `token.FileSet` instead of creating a new one per call.

```go
type CoreStructParser struct {
    // ... existing fields
    sharedFileSet *token.FileSet // NEW: reused across packages.Load calls
}

func (c *CoreStructParser) getOrCreateFileSet() *token.FileSet {
    if c.sharedFileSet == nil {
        c.sharedFileSet = token.NewFileSet()
    }
    return c.sharedFileSet
}
```

Replace `Fset: token.NewFileSet()` with `Fset: c.getOrCreateFileSet()` in `LookupStructFields`. Same pattern for `ParserEnumLookup`.

`token.FileSet` is already safe for concurrent use (internal mutex).

## Data Models

No new domain types. The only new types are orchestrator-local:

- `fileRoutes` struct — holds routes collected per file during parallel parsing (orchestrator-local)
- `map[string]bool` — set of referenced type names (ephemeral, passed from Phase 4 to Phase 5)

## Error Handling

### Route Ref Collection
- **No errors possible.** Pure function walking already-parsed route structures. Missing refs are simply not collected.

### Demand-Driven Schema Building
- **Type not found in registry:** Log warning and skip. This matches current behavior where annotations can reference types that don't exist in the loaded codebase.
- **Schema build error:** Return error to caller, same as current Phase 5.
- **Transitive resolution error:** Handled by existing `buildSchemasRecursive` error propagation.

### Parallel Route Parsing
- **File parse error:** Propagated via errgroup. First error cancels remaining goroutines via context.
- **No races:** Route parser has no mutable state. Results collected via mutex-protected slice.

### Cache Seeding
- **No errors possible.** Iterates already-loaded packages. Nil packages are skipped.

## Testing Strategy

### Correctness (Critical)
The single most important verification: **output must be identical.**

1. `make test-project-1` must produce output matching `testing/project-1-example-swagger.json`
2. `make test-project-2` must produce output matching `testing/project-2-example-swagger.json`
3. `TestRealProjectIntegration` must pass

These tests already exist and exercise the full pipeline end-to-end.

### Unit Tests

4. **Ref collector** (`internal/orchestrator/refs_test.go`):
   - Routes with no refs → empty set
   - Routes with body param refs → correct type names
   - Routes with response refs → correct type names
   - Routes with array/nested refs → recursively collected
   - Routes with Public refs → collected with suffix
   - Duplicate refs across routes → deduplicated

5. **Cache seeding** (`internal/model/struct_field_lookup_test.go`, `internal/model/enum_lookup_test.go`):
   - Seed with empty slice → no crash
   - Seed with packages → cache contains entries
   - Seed with nil packages in slice → skipped
   - Seed doesn't overwrite existing entries

6. **Registry FindTypeSpecByName** (`internal/registry/service_test.go`):
   - Known type → returns TypeSpecDef
   - Unknown type → returns nil

### Race Detection

7. `go test -race ./...` must pass — especially important for parallel route parsing and cache seeding.

### Benchmarks

8. Benchmark `Parse()` on test project data to measure before/after improvement.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Demand-driven misses types that eager approach caught | Medium | High | Both test projects have zero orphans — all defs are route-reachable. Integration tests catch any mismatch. |
| Public variant not built for some type | Medium | High | Collector captures `*Public` refs directly. `buildSchemasRecursive` builds Public variants for nested types. Integration tests verify. |
| Route annotation has unqualified type name | Low | Medium | Registry lookup falls back to searching packages. Same behavior as current Phase 5. |
| Parallel route parsing changes output order | Medium | Medium | Sort by file path before merging. Determinism verified by integration tests. |
| Cache seeding introduces stale data | Low | Low | Cache is additive. Fallback `packages.Load()` still works for cache misses. |

## Implementation Order

Each step can be verified independently via integration tests:

1. **Cache seeding** (P1) — standalone, no pipeline change, biggest reduction in `packages.Load()` calls
2. **Registry `FindTypeSpecByName`** (P0) — trivial addition, prerequisite for demand-driven building
3. **Route ref collector** (P0) — pure function, can be unit tested in isolation
4. **Pipeline reorder + demand-driven Phase 5** (P0) — the core change: remove Phase 3.5, reorder Phase 4 before Phase 5, wire up demand-driven building
5. **Parallel route parsing** (P1) — add errgroup to Phase 4
6. **Shared FileSet** (P2) — minor optimization

Steps 1-3 are safe to land independently. Step 4 is the big change that restructures the pipeline. Steps 5-6 are additive optimizations on top of the new pipeline.
