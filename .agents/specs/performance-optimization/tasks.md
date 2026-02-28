# Performance Optimization: Implementation Tasks

> **Context:** Read `requirements.md` and `design.md` in this directory for full architecture details.
> **Critical test:** `make test-project-1` and `make test-project-2` must produce identical output after every task.

---

## Task 1: Add Cache Seeding Functions

**Requirements:** 5.1, 5.2, 5.3, 5.4, 5.7, 5.8

**Implementation:**
1. In `internal/model/struct_field_lookup.go`, add `SeedGlobalPackageCache(pkgs []*packages.Package)` function after the `globalPackageCache` declaration (line ~22). It must:
   - Acquire `globalCacheMutex.Lock()`
   - Recursively walk all packages and their `pkg.Imports`
   - Store each `pkg` into `globalPackageCache[pkg.PkgPath]` if not already present
   - Use a `visited map[string]bool` to prevent infinite recursion on circular imports
   - Skip nil packages
2. In `internal/model/enum_lookup.go`, add `SeedEnumPackageCache(pkgs []*packages.Package)` function after the `enumPackageCache` declaration (line ~20). Same pattern as above but writing to `enumPackageCache`.
3. Write unit tests for both functions in their respective test files:
   - Seed with nil slice → no panic
   - Seed with empty slice → no entries added
   - Seed with packages → cache contains entries keyed by PkgPath
   - Seed with nil package in slice → skipped gracefully
   - Seed doesn't overwrite existing cache entries
   - Seed walks imports recursively

**Verification:**
- Run tests: `go test ./internal/model/ -run TestSeed -v`
- Expected: all tests pass
- Run: `go test -race ./internal/model/`
- Expected: no race conditions

**Self-Correction:**
- If tests fail: check mutex usage, ensure visited map prevents infinite loops
- If race detector fires: verify Lock/Unlock pairing

**Completion Criteria:**
- [ ] `SeedGlobalPackageCache` exists and is tested
- [ ] `SeedEnumPackageCache` exists and is tested
- [ ] `go test -race ./internal/model/` passes

**Escape Condition:** If stuck after 3 iterations, document the blocker and move to next task.

---

## Task 2: Wire Cache Seeding into Orchestrator

**Requirements:** 5.1, 5.5, 5.6, 5.8

**Implementation:**
1. In `internal/orchestrator/service.go`, in the `Parse()` method, add cache seeding immediately after the loading step (after line ~218, before Step 2). Add:
   ```go
   if loadResult.Packages != nil {
       model.SeedGlobalPackageCache(loadResult.Packages)
       model.SeedEnumPackageCache(loadResult.Packages)
   }
   ```
2. Add `"github.com/griffnb/core-swag/internal/model"` to the imports if not already present.
3. Add debug logging around the seeding call.

**Verification:**
- Run: `go build ./...`
- Expected: compiles cleanly
- Run: `go test ./internal/orchestrator/ -v`
- Expected: all existing tests pass
- Run: `make test-project-1` and `make test-project-2`
- Expected: identical output to before (cache seeding is additive, doesn't change behavior)

**Self-Correction:**
- If compilation fails: check import path matches the module path in go.mod
- If tests fail: the seeding should be purely additive — verify it doesn't overwrite existing entries

**Completion Criteria:**
- [ ] Cache seeding called in orchestrator after loading
- [ ] All existing tests pass
- [ ] `make test-project-1` output unchanged

**Escape Condition:** If stuck after 3 iterations, document the blocker and move to next task.

---

## Task 3: Add Registry FindTypeSpecByName

**Requirements:** 1.3, 1.8, 3.4

**Implementation:**
1. In `internal/registry/service.go`, add a new method:
   ```go
   // FindTypeSpecByName looks up a type definition by its qualified name
   // (e.g., "account.Account"). Returns nil if not found.
   func (s *Service) FindTypeSpecByName(name string) *domain.TypeSpecDef {
       return s.uniqueDefinitions[name]
   }
   ```
2. Write unit tests in `internal/registry/service_test.go`:
   - Look up a known type → returns correct TypeSpecDef
   - Look up unknown type → returns nil
   - Look up empty string → returns nil

**Verification:**
- Run tests: `go test ./internal/registry/ -run TestFindTypeSpecByName -v`
- Expected: all tests pass
- Run: `go test ./internal/registry/ -v`
- Expected: all existing tests still pass

**Self-Correction:**
- If the test setup is complex: look at existing test patterns in registry tests for how to populate the registry with test data
- If key format doesn't match: check `TypeSpecDef.TypeName()` in `internal/domain/types.go:55` and `parseTypesFromFile` in `internal/registry/types.go` to see how keys are generated

**Completion Criteria:**
- [ ] `FindTypeSpecByName` exists and is tested
- [ ] All existing registry tests pass

**Escape Condition:** If stuck after 3 iterations, document the blocker and move to next task.

---

## Task 4: Build Route Ref Collector

**Requirements:** 1.1, 1.2

**Implementation:**
1. Create new file `internal/orchestrator/refs.go` with:
   - `CollectReferencedTypes(routes []*routedomain.Route) map[string]bool` — walks all routes, collects unique type names from `$ref` strings
   - `collectRefsFromSchema(schema *routedomain.Schema, refs map[string]bool)` — recursively walks Schema structs extracting Ref values
2. The Schema struct (`internal/parser/route/domain/route.go:137`) has these fields to walk: `Ref`, `Items`, `Properties`, `AllOf`
3. Strip `#/definitions/` prefix from each Ref to get bare type names
4. Import `routedomain "github.com/griffnb/core-swag/internal/parser/route/domain"`
5. Write thorough unit tests in `internal/orchestrator/refs_test.go`:
   - Empty routes slice → empty map
   - Route with body param `$ref` → type name collected
   - Route with response `$ref` → type name collected
   - Route with array items `$ref` → nested ref collected
   - Route with AllOf `$ref` list → all refs collected
   - Route with Properties containing `$ref` → refs collected
   - Duplicate refs across routes → deduplicated (map)
   - Route with `#/definitions/account.AccountPublic` → `"account.AccountPublic"` in result
   - Route with no `$ref` (primitive types only) → empty map
   - Nil schema fields → no panic

**Verification:**
- Run tests: `go test ./internal/orchestrator/ -run TestCollectReferencedTypes -v`
- Expected: all tests pass
- Run: `go test -race ./internal/orchestrator/`
- Expected: no races (pure function, but verify)

**Self-Correction:**
- If import path wrong: check `internal/parser/route/domain/route.go` for the actual package name
- If Schema struct fields don't match: re-read the Schema struct definition at route/domain/route.go:137

**Completion Criteria:**
- [ ] `CollectReferencedTypes` exists in `refs.go`
- [ ] All test cases pass
- [ ] Handles all Schema nesting (Items, Properties, AllOf)

**Escape Condition:** If stuck after 3 iterations, document the blocker and move to next task.

---

## Task 5: Restructure Orchestrator Pipeline (Core Change)

**Requirements:** 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 2.1, 2.2, 2.3, 3.1, 3.2, 3.3, 3.4

This is the main pipeline restructure. The orchestrator `Parse()` method changes from eager to demand-driven.

**Implementation:**
1. In `internal/orchestrator/service.go`, modify the `Parse()` method:

   **a) Remove Phase 3.5** (lines 282-305): Delete the entire `structParser.ParseFile()` loop. Leave a comment noting it was removed in favor of demand-driven schema building.

   **b) Move route parsing before schema building** (already the case — Phase 4 is lines 307-360, Phase 5 is lines 366-403). The order is already correct after removing Phase 3.5. Just verify routes are parsed before schemas are built.

   **c) After route parsing, collect referenced types:** After the route parsing loop, add:
   ```go
   referencedTypes := CollectReferencedTypes(allRoutes)
   ```
   You'll need to accumulate `allRoutes []*routedomain.Route` during the route parsing loop.

   **d) Replace Phase 5 (lines 366-403)** with demand-driven schema building:
   - Iterate `referencedTypes` instead of `registry.UniqueDefinitions()`
   - For each type name:
     - Determine if it's a Public variant (ends with "Public")
     - Strip "Public" suffix to get base type name if needed
     - Look up via `s.registry.FindTypeSpecByName(baseTypeName)`
     - If not found, try the full name as-is (some types may be registered differently)
     - If still not found, log warning and skip
     - For struct types: use `CoreStructParser.BuildAllSchemas()` (struct_field_lookup.go:593) which handles base + Public + transitive dependencies and returns `map[string]*spec.Schema`
     - For non-struct types (aliases, enums): use `s.schemaBuilder.BuildSchema(typeDef)` which handles `*ast.Ident` and `*ast.SelectorExpr`
   - Add all resulting schemas to `s.swagger.Definitions`

2. Important: `BuildAllSchemas` (struct_field_lookup.go:593) signature is:
   ```go
   func (c *CoreStructParser) BuildAllSchemas(baseModule, importPath, typeName string, enumLookup TypeEnumLookup) (map[string]*spec.Schema, error)
   ```
   - `baseModule`: can be empty string `""`
   - `importPath`: the package path (e.g., `"github.com/user/project/internal/models/account"`)
   - `typeName`: the struct name (e.g., `"Account"`)
   - `enumLookup`: pass the enum lookup instance
   - Returns `map[string]*spec.Schema` containing all schemas including nested types

3. To get `importPath` and `typeName` from a qualified name like `"account.Account"`:
   - Split on last `.` to get package alias + type name
   - Look up the TypeSpecDef via `FindTypeSpecByName` which has `PkgPath` field for the full import path

4. Track which types have already been processed to avoid duplicates (a type might be both directly referenced and a transitive dependency of another type). Use a `processed map[string]bool`.

5. The orchestrator will need access to the `CoreStructParser` instance. It's currently stored as a field on `SchemaBuilder`. Either:
   - Add a getter on SchemaBuilder: `func (b *BuilderService) StructParser() *model.CoreStructParser`
   - Or store the CoreStructParser directly on the orchestrator Service struct (it's already created in `New()` at line 118)

**Verification:**
- Run: `go build ./...`
- Expected: compiles cleanly
- Run: `go test ./internal/orchestrator/ -v`
- Expected: all tests pass
- Run: `make test-project-1`
- Expected: **identical output** to `testing/project-1-example-swagger.json`
- Run: `make test-project-2`
- Expected: **identical output** to `testing/project-2-example-swagger.json`
- Run: `go test -race ./...`
- Expected: no races

**Self-Correction:**
- If output differs from expected: diff the JSON outputs carefully. Common issues:
  - Missing Public variants → ensure BuildAllSchemas generates them (it does at line 618)
  - Missing nested types → ensure BuildAllSchemas recursive resolution is working
  - Missing enum/alias schemas → ensure non-struct types still go through BuildSchema
  - Extra or missing definitions → check the processed map to avoid duplicates
- If a type can't be found by name: the uniqueDefinitions key format may differ from the route $ref format. Debug by logging both the route ref names and the registry keys.
- If BuildAllSchemas fails: check that the CoreStructParser has been properly initialized and the global cache has been seeded
- Update `.agents/change_log.md` with what was tried and what worked

**Completion Criteria:**
- [ ] Phase 3.5 removed from orchestrator
- [ ] Route parsing happens before schema building
- [ ] Only route-referenced types get schemas built
- [ ] `make test-project-1` produces identical output
- [ ] `make test-project-2` produces identical output
- [ ] `go test -race ./...` passes

**Escape Condition:** If output doesn't match after 3 iterations, add back Phase 3.5 as a fallback and document which types are missing. The cache seeding from Tasks 1-2 still provides value even without the pipeline restructure.

---

## Task 6: Parallel Route Parsing

**Requirements:** 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7

**Implementation:**
1. In `internal/orchestrator/service.go`, replace the sequential route parsing loop (Phase 4) with an `errgroup`-based parallel implementation.
2. Add imports: `"context"`, `"runtime"`, `"sort"`, `"sync"`, `"golang.org/x/sync/errgroup"`
3. Define a local `fileRoutes` struct to hold per-file results:
   ```go
   type fileRoutes struct {
       filePath string
       routes   []*routedomain.Route
   }
   ```
4. Use `errgroup.WithContext` + `g.SetLimit(runtime.NumCPU())` — follow the existing pattern in `internal/format/format.go:63`.
5. Each goroutine calls `s.routeParser.ParseRoutes(astFile, fileInfo.Path, fileInfo.FileSet)` and appends results to a mutex-protected `[]fileRoutes` slice.
6. After `g.Wait()`, sort `collected` by `filePath` for deterministic output.
7. Merge sorted results into `swagger.Paths` sequentially and accumulate `allRoutes` for ref collection.
8. The `routeParser.ParseRoutes` method has no mutable state — it's safe for concurrent per-file use. The only registry call it makes is `hasNoPublicAnnotation()` which is a read-only lookup. Registry writes are all done in Phase 2 before this point.

**Verification:**
- Run: `go build ./...`
- Expected: compiles cleanly
- Run: `go test ./internal/orchestrator/ -v`
- Expected: all tests pass
- Run: `make test-project-1`
- Expected: identical output (deterministic despite parallelism)
- Run: `make test-project-2`
- Expected: identical output
- Run: `go test -race ./...`
- Expected: no races

**Self-Correction:**
- If output order differs: verify the sort is correct — sort by `filePath` ascending
- If data race detected: check that no goroutine writes to shared state except the mutex-protected `collected` slice
- If route parsing errors in parallel but not sequential: check for shared state in routeParser that isn't visible (re-read `internal/parser/route/service.go` Service struct)

**Completion Criteria:**
- [ ] Route parsing uses errgroup with bounded concurrency
- [ ] Results sorted by file path for determinism
- [ ] `make test-project-1` and `make test-project-2` produce identical output
- [ ] `go test -race ./...` passes

**Escape Condition:** If races can't be resolved after 3 iterations, revert to sequential and document the shared state that caused the issue.

---

## Task 7: Shared FileSet for Fallback packages.Load

**Requirements:** 6.1, 6.2, 6.3

**Implementation:**
1. In `internal/model/struct_field_lookup.go`:
   - Add `sharedFileSet *token.FileSet` field to `CoreStructParser` struct (line ~24)
   - Add a `getOrCreateFileSet()` method that lazily initializes:
     ```go
     func (c *CoreStructParser) getOrCreateFileSet() *token.FileSet {
         if c.sharedFileSet == nil {
             c.sharedFileSet = token.NewFileSet()
         }
         return c.sharedFileSet
     }
     ```
   - Replace `Fset: token.NewFileSet()` at line 93 with `Fset: c.getOrCreateFileSet()`
2. In `internal/model/enum_lookup.go`:
   - Add `sharedFileSet *token.FileSet` field to `ParserEnumLookup` struct (line ~23)
   - Add same `getOrCreateFileSet()` method
   - At the `packages.Config` (line ~97), add `Fset: p.getOrCreateFileSet()`
3. `token.FileSet` is thread-safe internally — no additional mutex needed.

**Verification:**
- Run: `go test ./internal/model/ -v`
- Expected: all tests pass
- Run: `make test-project-1`
- Expected: identical output
- Run: `go test -race ./internal/model/`
- Expected: no races

**Self-Correction:**
- If tests fail with FileSet-related errors: verify that reusing a FileSet across multiple packages.Load calls doesn't cause position conflicts
- If it does cause issues: each packages.Load call may need its own FileSet, in which case this optimization isn't safe — revert and document

**Completion Criteria:**
- [ ] CoreStructParser reuses shared FileSet
- [ ] ParserEnumLookup reuses shared FileSet
- [ ] All tests pass, including integration

**Escape Condition:** If FileSet reuse causes issues, revert. This is a minor optimization — not worth risking correctness.

---

## Task 8: Add Debug Logging and Benchmarks

**Requirements:** 7.1, 7.2, 7.3, 7.4

**Implementation:**
1. In `internal/model/struct_field_lookup.go`, add atomic counters:
   ```go
   var (
       cacheHits   int64
       cacheMisses int64
   )
   ```
   - Increment `cacheHits` (via `atomic.AddInt64`) when `globalPackageCache` lookup succeeds in `LookupStructFields`
   - Increment `cacheMisses` when falling back to `packages.Load()`
   - Add `GlobalCacheStats() (hits, misses int64)` exported function
   - Add `ResetGlobalCacheStats()` for test isolation

2. In `internal/orchestrator/service.go`, after schema building completes, add debug logging:
   ```go
   if s.config.Debug != nil {
       hits, misses := model.GlobalCacheStats()
       s.config.Debug.Printf("Orchestrator: Package cache hits=%d misses=%d", hits, misses)
       s.config.Debug.Printf("Orchestrator: Built %d schemas from %d route-referenced types",
           len(s.swagger.Definitions), len(referencedTypes))
   }
   ```

3. Create `internal/orchestrator/benchmark_test.go` with a benchmark that measures `Parse()` on a test fixture. Use a small but representative set of files. The benchmark should:
   - Set up a minimal project structure with a few routes and structs
   - Call `Parse()` in a `b.N` loop
   - Report allocations via `b.ReportAllocs()`

**Verification:**
- Run: `go test ./internal/orchestrator/ -bench=. -benchmem`
- Expected: benchmark runs and reports timing + allocations
- Run: `go test ./internal/model/ -run TestGlobalCacheStats -v`
- Expected: stats increment correctly
- Run: `go test -race ./...`
- Expected: no races (atomic operations are race-safe)

**Self-Correction:**
- If benchmark is too slow or flaky: simplify the test fixture
- If atomic counters cause issues: they shouldn't, but verify import of `sync/atomic`

**Completion Criteria:**
- [ ] Cache hit/miss counters work and are tested
- [ ] Debug logging reports route-referenced type count and cache stats
- [ ] Benchmark test exists and runs
- [ ] `go test -race ./...` passes

**Escape Condition:** If benchmark setup is too complex, skip it — the cache stats and debug logging are the important parts.

---

## Task 9: Final Verification and Cleanup

**Requirements:** 1.7, 2.3, 7.1, 7.4

**Implementation:**
1. Run the full test suite: `go test ./... -v`
2. Run race detection: `go test -race ./...`
3. Run integration tests:
   - `make test-project-1` — diff output against `testing/project-1-example-swagger.json`
   - `make test-project-2` — diff output against `testing/project-2-example-swagger.json`
4. Run with debug mode enabled on a test project to verify the new logging output
5. Review all changed files for:
   - Unused imports
   - Dead code from Phase 3.5 removal (the orchestrator call is gone but StructParserService code stays)
   - Consistent error handling patterns
   - Go doc comments on all new exported functions
6. Update `.agents/change_log.md` with a summary of all changes made

**Verification:**
- Run: `go test ./... -v`
- Expected: all tests pass
- Run: `go test -race ./...`
- Expected: no races
- Run: `go vet ./...`
- Expected: no issues
- Run: `make test-project-1` and `make test-project-2`
- Expected: identical output to expected files

**Self-Correction:**
- If any test fails: fix the issue, don't skip
- If output differs: this is the most critical check — diff carefully and trace back to the specific type that's different

**Completion Criteria:**
- [ ] All unit tests pass
- [ ] All integration tests pass with identical output
- [ ] `go test -race ./...` clean
- [ ] `go vet ./...` clean
- [ ] Change log updated

**Escape Condition:** N/A — this is the final verification. All issues must be resolved.
