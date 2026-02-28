# Performance Optimization: Demand-Driven Pipeline

## Introduction

The core-swag orchestrator currently uses an **eager, supply-driven** pipeline: it parses ALL structs in ALL loaded files, builds schemas for ALL registry types, and only then parses routes — even though routes are the sole consumer of schema definitions. Routes produce `$ref` strings from comment annotations and never look up schemas during parsing. The result: the system does significant wasted work building schemas for types that may never be referenced.

Additionally, the most expensive operation — `packages.Load()` — is called redundantly because the loader's already-loaded packages are never shared downstream with the struct parser or enum lookup caches.

This feature restructures the pipeline into a **demand-driven** approach:
1. **Parse routes first** — discover which types are actually referenced
2. **Build only referenced schemas** — resolve types on demand, including transitive dependencies
3. **Seed caches** — share the loader's packages downstream to eliminate redundant `packages.Load()` calls
4. **Parallelize route parsing** — routes are independent per-file and can be processed concurrently

### Current Pipeline (Eager / Backwards)

```
Step 1:   Load ALL files + packages
Step 2:   Register ALL types into registry
Step 3:   Parse API info
Step 3.5: Parse EVERY struct → build schemas for ALL of them (wasteful)
Step 4:   Parse routes → produce $ref strings (never consults schemas)
Step 5:   Build schemas for ALL registry types → skips ones from 3.5
```

### Proposed Pipeline (Demand-Driven)

```
Step 1: Load ALL files + packages
Step 1b: Seed downstream caches from loaded packages
Step 2: Register ALL types into registry
Step 3: Parse API info
Step 4: Parse routes in parallel → collect all $ref type names
Step 5: Build schemas ONLY for route-referenced types (+ transitive deps)
```

Phase 3.5 (eager struct parsing) is eliminated entirely. Phase 5 becomes demand-driven.

---

## Requirements

### 1. Demand-Driven Schema Building

**As a** developer using core-swag,
**I want** schemas to be built only for types actually referenced by routes,
**So that** the system avoids wasted work parsing unreferenced types and runs faster.

**Acceptance Criteria:**

1.1. **When** route parsing completes, **the system shall** walk all parsed routes and collect every unique type name from `$ref` strings in parameters, responses, and nested schemas (Items, Properties, AllOf).

1.2. **The system shall** strip the `#/definitions/` prefix from collected refs to produce bare type names (e.g., `account.Account`, `account.AccountPublic`).

1.3. **When** the set of referenced type names is collected, **the system shall** build schemas only for those types using `CoreStructParser` and `SchemaBuilder`.

1.4. **When** building a schema for a referenced type, **the system shall** transitively resolve nested type dependencies (using the existing `nestedTypes` return from `StructField.ToSpecSchema()` and `buildSchemasRecursive()`).

1.5. **When** a referenced type name ends with `Public`, **the system shall** build the Public variant schema using the existing Public mode filtering (`public:"view|edit"` tags).

1.6. **When** a referenced type name does NOT end with `Public`, **the system shall** still build both the base AND Public variant if the Public variant is also referenced by routes.

1.7. **The system shall** produce identical Swagger output to the current eager pipeline — `make test-project-1` and `make test-project-2` must produce the same JSON.

1.8. **The system shall** handle edge cases where a type name from a route annotation doesn't exist in the registry (log warning, skip — matching current behavior).

---

### 2. Eliminate Phase 3.5 (Eager Struct Parsing)

**As a** developer,
**I want** the orchestrator-level StructParserService (`ParseFile` loop) to be removed from the pipeline,
**So that** there is a single schema-building code path and no redundant work.

**Acceptance Criteria:**

2.1. **The system shall** remove the Phase 3.5 loop that calls `structParser.ParseFile()` for every loaded file from the orchestrator's `Parse()` method.

2.2. **The system shall** ensure all functionality currently provided by Phase 3.5 is covered by the demand-driven Phase 5, specifically:
- Base struct schemas (handled by `CoreStructParser.LookupStructFields` + `BuildSpecSchema`)
- Public variant schemas (must be added to the demand-driven path)
- Embedded field resolution
- Enum field handling

2.3. **The system shall** continue to pass all existing tests without modification to test assertions.

2.4. **The system shall** NOT remove the `StructParserService` code itself — it may still be useful for other callers or future use. Only the orchestrator's call to it is removed.

---

### 3. Route-First Pipeline Reordering

**As a** developer,
**I want** routes to be parsed before any schema building occurs,
**So that** the system knows which types are needed before doing expensive schema work.

**Acceptance Criteria:**

3.1. **The system shall** reorder the orchestrator `Parse()` method so that route parsing (current Step 4) executes before schema building (current Step 5).

3.2. **The system shall** keep type registration (Step 2 + `ParseTypes()`) before route parsing, because `hasNoPublicAnnotation()` in the route parser reads from the registry.

3.3. **The system shall** keep API info parsing (Step 3) before route parsing (no dependency, but maintains logical order).

3.4. **The system shall** pass the collected set of referenced type names from route parsing to the schema building phase.

---

### 4. Parallel Route Parsing

**As a** developer using core-swag on large projects,
**I want** route parsing to process files concurrently,
**So that** multi-core CPUs are utilized during the route discovery phase.

**Acceptance Criteria:**

4.1. **When** the orchestrator enters the route parsing phase, **the system shall** process files concurrently using a bounded worker pool.

4.2. **The system shall** use `errgroup` (from `golang.org/x/sync/errgroup`, already in go.mod) to manage concurrent goroutines with error propagation.

4.3. **The system shall** limit concurrency to `runtime.NumCPU()` workers by default.

4.4. **When** routes are parsed concurrently, **the system shall** collect results into per-goroutine slices and merge them after all goroutines complete — no concurrent writes to shared maps.

4.5. **The system shall** produce deterministic output by sorting collected routes by file path before merging into `swagger.Paths`.

4.6. **The system shall** not introduce any data races — all concurrent code must pass `go test -race`.

4.7. **The system shall** keep file registration (`CollectAstFile`) sequential because it mutates the non-thread-safe registry.

---

### 5. Cache Seeding from Loader

**As a** developer using core-swag,
**I want** the initial loader's package data to be shared with downstream caches,
**So that** redundant `packages.Load()` calls are eliminated during demand-driven schema building.

**Acceptance Criteria:**

5.1. **When** `LoadWithGoPackages()` completes, **the system shall** extract all `*packages.Package` objects from the loaded result and recursively walk their imports.

5.2. **The system shall** seed the `globalPackageCache` in `struct_field_lookup.go` with each package keyed by its import path.

5.3. **The system shall** seed the `enumPackageCache` in `enum_lookup.go` with each package keyed by its import path.

5.4. **When** a cache is seeded, **the system shall** use write locks (existing `sync.RWMutex`) to ensure thread-safe population.

5.5. **When** `CoreStructParser.LookupStructFields()` is called for a type during demand-driven schema building, **the system shall** find the package in the pre-seeded global cache and avoid calling `packages.Load()`.

5.6. **When** `ParserEnumLookup.GetEnumsForType()` is called for a type, **the system shall** find the package in the pre-seeded enum cache and avoid calling `packages.Load()`.

5.7. **When** a package is NOT found in the pre-seeded cache (e.g., external dependency not in initial load), **the system shall** fall back to calling `packages.Load()` — the cache is additive, not a hard gate.

5.8. **The system shall** not break existing behavior — all existing tests must continue to pass.

---

### 6. Shared FileSet Across packages.Load Calls

**As a** developer,
**I want** fallback `packages.Load()` calls to reuse a shared `token.FileSet`,
**So that** memory allocation and GC pressure are reduced.

**Acceptance Criteria:**

6.1. **When** `CoreStructParser.LookupStructFields()` falls back to `packages.Load()`, **the system shall** reuse a shared `token.FileSet` stored on the parser instance rather than creating `token.NewFileSet()` each time.

6.2. **When** `ParserEnumLookup` falls back to `packages.Load()`, **the system shall** reuse a shared `token.FileSet`.

6.3. **The system shall** ensure the shared FileSet is thread-safe (Go's `token.FileSet` is safe for concurrent use via its internal mutex).

---

### 7. Observability & Verification

**As a** developer,
**I want** to verify that the demand-driven pipeline produces correct output and is faster,
**So that** I can confirm the changes work.

**Acceptance Criteria:**

7.1. **The system shall** verify correctness via `make test-project-1` and `make test-project-2` producing identical output.

7.2. **The system shall** add optional debug logging that reports:
- Number of types referenced by routes
- Number of schemas built (should equal referenced types + transitive deps)
- Cache hit/miss ratios for global package cache and enum cache

7.3. **The system shall** include benchmark tests comparing parse time before and after.

7.4. **The system shall** pass `go test -race ./...`.

---

## Priority Order

| Priority | Requirement | Impact | Risk | Effort |
|----------|------------|--------|------|--------|
| **P0** | 1. Demand-Driven Schema Building | Very High (skip all unreferenced types) | Medium | Medium |
| **P0** | 2. Eliminate Phase 3.5 | High (removes redundant code path) | Medium | Low |
| **P0** | 3. Route-First Pipeline Reorder | High (enables demand-driven) | Low | Low |
| **P1** | 5. Cache Seeding from Loader | High (eliminates redundant `packages.Load()`) | Low | Medium |
| **P1** | 4. Parallel Route Parsing | Medium-High (multi-core utilization) | Medium | Medium |
| **P2** | 6. Shared FileSet | Low-Medium (reduces GC pressure) | Low | Low |
| **P2** | 7. Observability & Verification | N/A (supporting) | None | Low |

## Out of Scope

- Parallelizing schema building — schemas have transitive dependencies that make safe parallelism complex
- Parallelizing file registration — registry is not thread-safe and rewriting it is not justified
- Changing `go/packages` load modes — could break type resolution
- Removing the `StructParserService` code — only removing the orchestrator's call to it
- Pruning the registry to only contain route-referenced types — registry is cheap and used for type resolution during schema building
