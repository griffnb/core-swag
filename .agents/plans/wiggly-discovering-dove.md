# Schema Building Performance Optimization

## Context

`buildDemandDrivenSchemas` takes 60+ seconds even after adding errgroup concurrency. The parallelism is ineffective because the real bottleneck is **inside** each `BuildAllSchemas` call — specifically `packages.Load` (spawns `go list` subprocess, 1-5s each). Integration tests show 142 cache misses = 142 subprocess spawns. Four root causes identified, four targeted fixes.

## Files to Modify

- `internal/model/struct_field_lookup.go` — Fixes 1, 2, 4
- `internal/orchestrator/schema_builder.go` — Fix 3, Fix 4 integration
- `internal/model/struct_field_lookup_test.go` — New tests for singleflight + signature updates

## Fix 1: Add singleflight to `packages.Load` (HIGH IMPACT)

**Problem:** Multiple goroutines call `packages.Load(cfg, importPath)` for the same package simultaneously (lines 195-237). No deduplication — all spawn separate `go list` subprocesses.

**Changes to `struct_field_lookup.go`:**

1. Add import: `"golang.org/x/sync/singleflight"`
2. Add package-level var alongside existing globals (line ~22):
   ```go
   var packageLoadGroup singleflight.Group
   ```
3. Wrap the `packages.Load` call (lines 195-237) in `packageLoadGroup.Do(importPath, func() ...)`:
   - The callback loads the package, builds the recursive packageMap, and caches everything in `globalPackageCache`
   - Use `token.NewFileSet()` inside the callback (not `c.getOrCreateFileSet()`) since the result may be shared across parser instances
   - On return, extract `pkg` and `packageMap` from the singleflight result
   - Populate `c.packageCache` (local) from the result outside the callback
4. Replace the `log.Fatalf` with returned error from singleflight (preserve fatal behavior)

**Expected impact:** Eliminates ~100+ duplicate `packages.Load` calls. 40-60% speedup.

## Fix 2: Replace full map copy with read-through accessor (MEDIUM IMPACT)

**Problem:** On every cache **hit** (lines 238-248), the entire `globalPackageCache` is copied into a new local `packageMap`. With thousands of entries, this is O(N) allocation per call across 3,737 hits. The copy exists because `checkNamed` (line 618) and `ExtractFieldsRecursive` (line 624) read from `c.packageMap`.

**Changes to `struct_field_lookup.go`:**

1. Add new method:
   ```go
   func (c *CoreStructParser) resolvePackage(pkgPath string) *packages.Package
   ```
   Checks `c.packageCache` first (local, no lock needed), then `globalPackageCache` (behind `RLock`).

2. Remove `packageMap` field from `CoreStructParser` struct (line ~99)

3. Remove the full map copy on cache hit (lines 238-248) — replace with just the atomic hit counter + debug log

4. Remove `c.packageMap = packageMap` (line 251) — the cache-miss path already populates `c.packageCache` at lines 229-235

5. Update `checkNamed` (line 618):
   ```go
   // Old: nextPackage, ok := c.packageMap[pkg.Path()]
   // New: nextPackage := c.resolvePackage(pkg.Path())
   ```

6. Update `processStructField` (line ~387) to use `c.resolvePackage()` instead of the `packageMap` parameter

7. Remove the unused `packageMap` parameter from `processStructField` and `ExtractFieldsRecursive` (the latter already uses `_`)

**Expected impact:** Eliminates O(N) map copies on every cache hit. 5-10% speedup.

## Fix 3: Pre-warm packages with Syntax before concurrent builds (HIGH IMPACT)

**Problem:** `SeedGlobalPackageCache` seeds transitive deps without AST Syntax (`go/packages` only populates Syntax for directly-requested packages). `LookupStructFields` rejects syntax-less entries (line 188-191), triggering a fresh `packages.Load`. This is the source of most of the 142 cache misses.

**Changes to `schema_builder.go`:**

1. Add new function `preWarmPackages(work []structRefWork, debug Debugger) error`:
   - Collect unique `pkgPath` values from the work slice
   - Filter out any already cached with Syntax in `globalPackageCache`
   - Call `packages.Load(cfg, pkg1, pkg2, ...)` with ALL paths in a single batched call — this triggers one `go list` invocation that resolves everything, dramatically faster than N individual calls
   - Seed results via `model.SeedGlobalPackageCache(pkgs)` and `model.SeedEnumPackageCache(pkgs)`

2. In `buildDemandDrivenSchemas`, insert call between Phase 1 (partition) and Phase 2 (concurrent build):
   ```go
   // Phase 1.5: Pre-warm packages with Syntax in a single batched call.
   if err := preWarmPackages(structWork, s.config.Debug); err != nil {
       // Non-fatal: concurrent builds fall back to individual loads (deduplicated by singleflight)
   }
   ```

3. Add new export to `struct_field_lookup.go`:
   ```go
   func IsPackageCachedWithSyntax(importPath string) bool
   ```
   So the pre-warm function can filter already-cached packages.

**Expected impact:** Single batched `go list` replaces N sequential ones. 20-30% speedup.

## Fix 4: Share typeCache across goroutines (LOW-MEDIUM IMPACT)

**Problem:** Each `BuildAllSchemas` creates `&CoreStructParser{}` with empty `typeCache` (line 710). If goroutine A and B both need `constants.Role`, both independently look it up.

**Changes to `struct_field_lookup.go`:**

1. Add new exported type:
   ```go
   type SharedTypeCache struct {
       types map[string]*StructBuilder
       mu    sync.RWMutex
   }
   func NewSharedTypeCache() *SharedTypeCache { ... }
   ```

2. Add optional `sharedCache *SharedTypeCache` field to `CoreStructParser`

3. In `LookupStructFields`, check shared cache before local cache (at line ~142):
   - Read from shared cache under `RLock`
   - On local cache store (line ~280), also store in shared cache under `Lock`

4. Add `BuildAllSchemasWithCache` function (or add optional `cache` parameter):
   ```go
   func BuildAllSchemasWithCache(baseModule, pkgPath, typeName string, cache *SharedTypeCache, packageNameOverride ...string) (map[string]*spec.Schema, error)
   ```
   Creates `&CoreStructParser{sharedCache: cache}` instead of bare `&CoreStructParser{}`

**Changes to `schema_builder.go`:**

5. In `buildStructSchemasConcurrent`, create one `SharedTypeCache` and pass it to all goroutines:
   ```go
   sharedCache := model.NewSharedTypeCache()
   // ... in each g.Go:
   schemas, err := model.BuildAllSchemasWithCache("", w.pkgPath, w.typeName, sharedCache, w.goPackageName)
   ```

**Thread safety:** Each goroutine still gets its own `CoreStructParser` (no races on `basePackage`, `visited`). Only the `SharedTypeCache` is shared, protected by its own `sync.RWMutex`. `StructBuilder` is immutable after creation, so sharing cached values is safe.

**Expected impact:** Avoids redundant type parsing across goroutines. 5-10% speedup.

## Implementation Order

1. **Fix 1** (singleflight) — highest impact, zero dependencies on other fixes
2. **Fix 3** (pre-warm) — reduces cache misses before concurrent phase
3. **Fix 2** (read-through) — eliminates map copy overhead on remaining hits
4. **Fix 4** (shared cache) — incremental benefit on top of others

## Verification

After each fix:
1. `go build ./internal/model/... ./internal/orchestrator/...` — compiles
2. `go test ./internal/model/... ./internal/orchestrator/...` — unit tests pass
3. `go test -race ./internal/model/... ./internal/orchestrator/...` — no data races
4. `make test-project-1` and `make test-project-2` — integration tests produce correct output
5. Check debug log for `Package cache hits/misses` — misses should drop from ~142 toward single digits

After all fixes:
- Total schema build time should drop from 60+ seconds to 10-20 seconds
