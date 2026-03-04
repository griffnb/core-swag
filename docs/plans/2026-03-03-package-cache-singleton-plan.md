# PackageCache Singleton Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace scattered global package cache variables with a `PackageCache` singleton struct that provides a single `GetOrLoad` method, eliminating non-deterministic schema generation.

**Architecture:** A `PackageCache` struct encapsulates the cache map, mutex, singleflight group, and stats counters. A `sync.Once`-guarded `Cache()` function returns the singleton. All callers use `Cache().GetOrLoad(pkgPath)` instead of `resolvePackage` or inline singleflight logic. The pre-warm batch load (pass 1 only) remains as a performance optimization via `Cache().Seed()`.

**Tech Stack:** Go stdlib (`sync`, `sync/atomic`, `go/token`), `golang.org/x/sync/singleflight`, `golang.org/x/tools/go/packages`

---

### Task 1: Create the PackageCache struct and singleton

**Files:**
- Create: `internal/model/package_cache.go`
- Test: `internal/model/package_cache_test.go`

**Step 1: Write the failing tests**

Create `internal/model/package_cache_test.go`:

```go
package model

import (
	"go/ast"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestCache_ReturnsSingleton(t *testing.T) {
	a := Cache()
	b := Cache()
	assert.Same(t, a, b, "Cache() should return the same instance")
}

func TestPackageCache_Seed_AddsEntries(t *testing.T) {
	c := newTestCache()

	pkgA := &packages.Package{PkgPath: "example.com/a", Syntax: []*ast.File{{}}}
	pkgB := &packages.Package{PkgPath: "example.com/b", Syntax: []*ast.File{{}}}
	c.Seed([]*packages.Package{pkgA, pkgB})

	assert.Same(t, pkgA, c.get("example.com/a"))
	assert.Same(t, pkgB, c.get("example.com/b"))
}

func TestPackageCache_Seed_WalksTransitiveImports(t *testing.T) {
	c := newTestCache()

	leaf := &packages.Package{PkgPath: "example.com/leaf", Syntax: []*ast.File{{}}}
	root := &packages.Package{
		PkgPath: "example.com/root",
		Syntax:  []*ast.File{{}},
		Imports: map[string]*packages.Package{"example.com/leaf": leaf},
	}
	c.Seed([]*packages.Package{root})

	assert.Same(t, root, c.get("example.com/root"))
	assert.Same(t, leaf, c.get("example.com/leaf"))
}

func TestPackageCache_Seed_HandlesCircularImports(t *testing.T) {
	c := newTestCache()

	pkgA := &packages.Package{PkgPath: "example.com/a", Imports: map[string]*packages.Package{}}
	pkgB := &packages.Package{PkgPath: "example.com/b", Imports: map[string]*packages.Package{"example.com/a": pkgA}}
	pkgA.Imports["example.com/b"] = pkgB

	assert.NotPanics(t, func() { c.Seed([]*packages.Package{pkgA}) })
	assert.NotNil(t, c.get("example.com/a"))
	assert.NotNil(t, c.get("example.com/b"))
}

func TestPackageCache_Seed_NilAndEmpty(t *testing.T) {
	c := newTestCache()
	assert.NotPanics(t, func() { c.Seed(nil) })
	assert.NotPanics(t, func() { c.Seed([]*packages.Package{}) })
	assert.NotPanics(t, func() { c.Seed([]*packages.Package{nil}) })
}

func TestPackageCache_Seed_DirectOverwritesTransitive(t *testing.T) {
	c := newTestCache()

	// Transitive dep (no syntax) gets cached first
	transitive := &packages.Package{PkgPath: "example.com/a", Name: "transitive"}
	c.Seed([]*packages.Package{
		{PkgPath: "example.com/root", Syntax: []*ast.File{{}},
			Imports: map[string]*packages.Package{"example.com/a": transitive}},
	})

	// Direct load with syntax should overwrite
	direct := &packages.Package{PkgPath: "example.com/a", Name: "direct", Syntax: []*ast.File{{}}}
	c.Seed([]*packages.Package{direct})

	assert.Equal(t, "direct", c.get("example.com/a").Name)
}

func TestPackageCache_IsCached_WithSyntax(t *testing.T) {
	c := newTestCache()

	withSyntax := &packages.Package{PkgPath: "example.com/a", Syntax: []*ast.File{{}}}
	noSyntax := &packages.Package{PkgPath: "example.com/b"}
	c.Seed([]*packages.Package{withSyntax, noSyntax})

	assert.True(t, c.IsCached("example.com/a"))
	assert.False(t, c.IsCached("example.com/b"))
	assert.False(t, c.IsCached("example.com/nonexistent"))
}

func TestPackageCache_Stats(t *testing.T) {
	c := newTestCache()
	c.ResetStats()

	hits, misses := c.Stats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)
}

func TestPackageCache_Get_ReturnsNilForMissing(t *testing.T) {
	c := newTestCache()
	assert.Nil(t, c.get("example.com/nonexistent"))
}

func TestPackageCache_ConcurrentSeed(t *testing.T) {
	c := newTestCache()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			pkg := &packages.Package{PkgPath: "example.com/concurrent", Syntax: []*ast.File{{}}}
			c.Seed([]*packages.Package{pkg})
		}(i)
	}
	wg.Wait()
	assert.NotNil(t, c.get("example.com/concurrent"))
}

// newTestCache creates a fresh PackageCache for test isolation (not the singleton).
func newTestCache() *PackageCache {
	return &PackageCache{
		packages: make(map[string]*packages.Package),
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/model/ -run TestCache -v -count=1`
Expected: FAIL — `PackageCache` type does not exist

**Step 3: Write the PackageCache struct**

Create `internal/model/package_cache.go`:

```go
package model

import (
	"fmt"
	"go/token"
	"sync"
	"sync/atomic"

	"github.com/griffnb/core-swag/internal/console"
	"golang.org/x/sync/singleflight"
	"golang.org/x/tools/go/packages"
)

// PackageCache is the central package resolution system. It maintains a cache
// of loaded Go packages and provides GetOrLoad to obtain a *packages.Package
// with full type information. All package lookups in the codebase should go
// through Cache().GetOrLoad(pkgPath).
type PackageCache struct {
	mu        sync.RWMutex
	packages  map[string]*packages.Package
	loadGroup singleflight.Group
	hits      int64
	misses    int64
}

var (
	cacheOnce     sync.Once
	cacheInstance *PackageCache
)

// Cache returns the singleton PackageCache instance.
func Cache() *PackageCache {
	cacheOnce.Do(func() {
		cacheInstance = &PackageCache{
			packages: make(map[string]*packages.Package),
		}
	})
	return cacheInstance
}

// get returns the cached package for pkgPath, or nil if not cached.
// Does not load — use GetOrLoad for the full get-or-load path.
func (c *PackageCache) get(pkgPath string) *packages.Package {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.packages[pkgPath]
}

// GetOrLoad returns a *packages.Package with Syntax for the given import path.
// Checks the cache first. On miss, loads via singleflight (deduplicating
// concurrent loads for the same path), caches the result, and returns it.
// Returns nil only when the package genuinely cannot be loaded.
func (c *PackageCache) GetOrLoad(pkgPath string) *packages.Package {
	// 1. Cache hit — package exists and has usable AST.
	c.mu.RLock()
	if pkg := c.packages[pkgPath]; pkg != nil && len(pkg.Syntax) > 0 {
		c.mu.RUnlock()
		atomic.AddInt64(&c.hits, 1)
		return pkg
	}
	c.mu.RUnlock()

	// 2. Cache miss — load via singleflight.
	atomic.AddInt64(&c.misses, 1)
	console.Logger.Debug("PackageCache: loading %s\n", pkgPath)

	val, err, _ := c.loadGroup.Do(pkgPath, func() (any, error) {
		cfg := &packages.Config{
			Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
				packages.NeedName | packages.NeedImports | packages.NeedDeps,
			Fset: token.NewFileSet(),
		}
		pkgs, err := packages.Load(cfg, pkgPath)
		if err != nil || len(pkgs) == 0 {
			return nil, fmt.Errorf("failed to load package %s: %v", pkgPath, err)
		}

		// Cache direct target and transitive deps.
		c.mu.Lock()
		c.seedLocked(pkgs)
		c.mu.Unlock()

		console.Logger.Debug("PackageCache: loaded %s\n", pkgPath)
		return pkgs[0], nil
	})
	if err != nil {
		console.Logger.Debug("PackageCache: failed to load %s: %v\n", pkgPath, err)
		return nil
	}
	return val.(*packages.Package)
}

// IsCached reports whether pkgPath has a usable entry (with Syntax) in the cache.
// Used by pre-warm to skip already-loaded packages.
func (c *PackageCache) IsCached(pkgPath string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pkg := c.packages[pkgPath]
	return pkg != nil && len(pkg.Syntax) > 0
}

// Seed bulk-populates the cache from a packages.Load result. Direct packages
// (in the pkgs slice) overwrite existing entries. Transitive imports are only
// cached if no existing entry exists (to avoid overwriting a Syntax-bearing
// entry with a Syntax-less transitive dep).
func (c *PackageCache) Seed(pkgs []*packages.Package) {
	if len(pkgs) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seedLocked(pkgs)
}

// seedLocked is the lock-free inner implementation of Seed. Caller must hold c.mu.
func (c *PackageCache) seedLocked(pkgs []*packages.Package) {
	visited := make(map[string]bool)

	// Pass 1: Direct packages always overwrite — they come from packages.Load
	// with full Syntax and TypesInfo.
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		c.packages[pkg.PkgPath] = pkg
		visited[pkg.PkgPath] = true
	}

	// Pass 2: Walk transitive imports. Only cache if no existing entry to avoid
	// overwriting a Syntax-bearing version with a Syntax-less transitive dep.
	var walk func(pkg *packages.Package)
	walk = func(pkg *packages.Package) {
		if pkg == nil || visited[pkg.PkgPath] {
			return
		}
		visited[pkg.PkgPath] = true
		if c.packages[pkg.PkgPath] == nil {
			c.packages[pkg.PkgPath] = pkg
		}
		for _, imp := range pkg.Imports {
			walk(imp)
		}
	}
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, imp := range pkg.Imports {
			walk(imp)
		}
	}
}

// Stats returns the cache hit and miss counts.
func (c *PackageCache) Stats() (hits, misses int64) {
	return atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses)
}

// ResetStats zeroes the hit/miss counters. Used for test isolation.
func (c *PackageCache) ResetStats() {
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
}

// Reset clears the entire cache and resets stats. Used for test isolation only.
func (c *PackageCache) Reset() {
	c.mu.Lock()
	c.packages = make(map[string]*packages.Package)
	c.mu.Unlock()
	c.ResetStats()
}

// Packages returns a snapshot of the cache map. Used by resolvePackagePath
// and buildSchemasRecursive for searching by package name. The returned map
// must NOT be modified by callers.
func (c *PackageCache) Packages() map[string]*packages.Package {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.packages
}

// RLock/RUnlock expose read-lock for callers that need to iterate the cache
// safely (e.g., resolvePackagePath, buildSchemasRecursive candidate search).
func (c *PackageCache) RLock()   { c.mu.RLock() }
func (c *PackageCache) RUnlock() { c.mu.RUnlock() }
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/model/ -run "TestCache|TestPackageCache" -v -count=1`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/model/package_cache.go internal/model/package_cache_test.go
git commit -m "feat: add PackageCache singleton struct with GetOrLoad, Seed, IsCached"
```

---

### Task 2: Migrate struct_field_lookup.go to use PackageCache

This is the core migration. Replace all bare global variables, `resolvePackage`, singleflight logic in `LookupStructFields`, `forceReloadPackage`, `hasASTStructFields`, and the `fullyLoaded` system with calls to `Cache()`.

**Files:**
- Modify: `internal/model/struct_field_lookup.go`
- Modify: `internal/model/struct_field_lookup_test.go`

**Step 1: Rewrite struct_field_lookup.go**

Starting from the **committed HEAD** version (not the working copy), make these changes:

**a) Remove all bare global cache variables and functions.** Delete:
- `globalPackageCache`, `fullyLoaded`, `globalCacheMutex`, `packageLoadGroup` declarations
- `cacheHits`, `cacheMisses` declarations
- `GlobalCacheStats()`, `ResetGlobalCacheStats()` functions
- `IsPackageCachedWithSyntax()` function
- `NotFullyLoadedPaths()` function
- `SeedGlobalPackageCache()` function
- `resolvePackage()` method on CoreStructParser
- `forceReloadPackage()` method on CoreStructParser
- `hasASTStructFields()` function

**b) Add thin wrappers** that delegate to `Cache()` so callers outside the package don't need to change yet:

```go
// SeedGlobalPackageCache delegates to Cache().Seed for backward compatibility.
func SeedGlobalPackageCache(pkgs []*packages.Package) {
	Cache().Seed(pkgs)
}

// IsPackageCachedWithSyntax delegates to Cache().IsCached for backward compatibility.
func IsPackageCachedWithSyntax(importPath string) bool {
	return Cache().IsCached(importPath)
}

// GlobalCacheStats delegates to Cache().Stats for backward compatibility.
func GlobalCacheStats() (hits, misses int64) {
	return Cache().Stats()
}

// ResetGlobalCacheStats delegates to Cache().ResetStats for backward compatibility.
func ResetGlobalCacheStats() {
	Cache().ResetStats()
}
```

**c) Remove `packageCache` field from `CoreStructParser`** (already removed in working copy). The struct becomes:

```go
type CoreStructParser struct {
	basePackage   *packages.Package
	visited       map[string]bool
	typeCache     map[string]*StructBuilder
	cacheMutex    sync.RWMutex
	sharedFileSet *token.FileSet
	sharedCache   *SharedTypeCache
}
```

**d) Rewrite `LookupStructFields`** to use `Cache().GetOrLoad()`. The entire 80+ line cache check / singleflight / retry block becomes:

```go
func (c *CoreStructParser) LookupStructFields(_, importPath, typeName string) *StructBuilder {
	cacheKey := importPath + ":" + typeName

	// Check type caches first — shared (cross-goroutine), then local.
	if c.sharedCache != nil {
		if cached, ok := c.sharedCache.get(cacheKey); ok {
			return cached
		}
	}
	c.cacheMutex.RLock()
	if cached, exists := c.typeCache[cacheKey]; exists {
		c.cacheMutex.RUnlock()
		return cached
	}
	c.cacheMutex.RUnlock()

	builder := &StructBuilder{}

	c.cacheMutex.Lock()
	if c.typeCache == nil {
		c.typeCache = make(map[string]*StructBuilder)
	}
	c.cacheMutex.Unlock()

	// Resolve the package — try exact path first, then suffix match for relative paths.
	pkg := Cache().GetOrLoad(importPath)
	if pkg == nil {
		// Try suffix match: importPath may be relative ("design/controllers/foo")
		suffix := "/" + importPath
		Cache().RLock()
		for fullPath, cachedPkg := range Cache().Packages() {
			if strings.HasSuffix(fullPath, suffix) && cachedPkg != nil && len(cachedPkg.Syntax) > 0 {
				pkg = cachedPkg
				importPath = fullPath
				cacheKey = importPath + ":" + typeName
				break
			}
		}
		Cache().RUnlock()
	}

	if pkg == nil || pkg.PkgPath != importPath {
		return builder
	}

	c.basePackage = pkg
	visited := make(map[string]bool)
	c.visited = visited
	fields := c.ExtractFieldsRecursive(pkg, typeName, visited)

	for _, f := range fields {
		if f.IsGeneric() && strings.Contains(f.EffectiveTypeString(), "fields.StructField") {
			c.processStructField(f, builder)
		} else {
			builder.Fields = append(builder.Fields, f)
		}
	}

	c.cacheMutex.Lock()
	c.typeCache[cacheKey] = builder
	c.cacheMutex.Unlock()
	if c.sharedCache != nil {
		c.sharedCache.set(cacheKey, builder)
	}

	return builder
}
```

**e) Update `checkNamed`** to use `Cache().GetOrLoad()` instead of `resolvePackage`:

```go
// In checkNamed, replace:
//   nextPackage := c.resolvePackage(pkg.Path())
// With:
nextPackage := Cache().GetOrLoad(pkg.Path())
```

**f) Update `processStructField`** to use `Cache().GetOrLoad()` instead of `resolvePackage`:

```go
// In processStructField, replace:
//   targetPkg := c.resolvePackage(subTypePackage)
// With:
targetPkg := Cache().GetOrLoad(subTypePackage)
```

**g) Keep the TypesInfo nil guard** in `ExtractFieldsRecursive` (line ~636 in working copy) — this is a genuine safety check:

```go
if pkg.TypesInfo == nil {
	console.Logger.Debug("Skipping field %s: pkg.TypesInfo is nil for %s\n", fieldName, pkg.PkgPath)
	continue
}
```

**Step 2: Rewrite struct_field_lookup_test.go**

Update test helpers and tests:

- Replace `resetGlobalPackageCache()` with a helper that calls `Cache().Reset()` (or use `newTestCache()` for isolated tests)
- Remove `TestSeedGlobalPackageCache_MarksFullyLoaded` — `fullyLoaded` concept is gone
- Remove `TestResolvePackage_ReturnsFullyLoaded`, `TestResolvePackage_SkipsNonFullyLoaded` — `resolvePackage` is gone
- Update `TestResolvePackage_ReturnsNilWhenNotFound` — not needed, `GetOrLoad` handles this
- Update `TestIsPackageCachedWithSyntax_*` tests to work with `Cache().IsCached()`
- Update `TestGlobalCacheStats_IncrementPattern` — seed needs Syntax (which it already has)
- Tests that directly manipulate `globalPackageCache` and `globalCacheMutex` should use `Cache().Seed()` or `newTestCache()` instead
- Keep `TestSharedTypeCache_*` and `TestSingleflight_*` tests as-is (they test different things)

**Step 3: Run tests**

Run: `go test ./internal/model/ -v -count=1 -race`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/model/struct_field_lookup.go internal/model/struct_field_lookup_test.go
git commit -m "refactor: migrate struct_field_lookup to PackageCache singleton

Replace resolvePackage, singleflight block, forceReloadPackage, fullyLoaded
system with Cache().GetOrLoad(). checkNamed and processStructField now always
load missing packages instead of silently dropping fields."
```

---

### Task 3: Migrate package_resolve.go to use PackageCache

**Files:**
- Modify: `internal/model/package_resolve.go`
- Modify: `internal/model/package_resolve_test.go`

**Step 1: Update package_resolve.go**

Replace direct access to `globalPackageCache` and `globalCacheMutex` with `Cache()` methods:

```go
func resolvePackagePath(parentPkgPath, shortName string) string {
	siblingPath := shortName
	if idx := strings.LastIndex(parentPkgPath, "/"); idx >= 0 {
		siblingPath = parentPkgPath[:idx+1] + shortName
	}

	// Fast path: sibling exists in the cache
	if Cache().get(siblingPath) != nil {
		return siblingPath
	}

	// Search cache for packages whose .Name matches shortName
	best := searchCacheByName(Cache(), parentPkgPath, shortName)
	if best != "" {
		return best
	}

	// Also search the enum package cache
	best = searchEnumCacheByName(parentPkgPath, shortName)
	if best != "" {
		return best
	}

	return siblingPath
}
```

Update `searchCacheByName` to accept `*PackageCache` instead of raw map + mutex:

```go
func searchCacheByName(cache *PackageCache, parentPkgPath, shortName string) string {
	cache.RLock()
	defer cache.RUnlock()

	var bestPath string
	bestPrefix := -1
	for pkgPath, pkg := range cache.Packages() {
		if pkg != nil && pkg.Name == shortName {
			pLen := commonPrefixLength(parentPkgPath, pkgPath)
			if pLen > bestPrefix {
				bestPrefix = pLen
				bestPath = pkgPath
			}
		}
	}
	return bestPath
}
```

Add a separate `searchEnumCacheByName` for the enum cache (which remains separate):

```go
func searchEnumCacheByName(parentPkgPath, shortName string) string {
	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()

	var bestPath string
	bestPrefix := -1
	for pkgPath, pkg := range enumPackageCache {
		if pkg != nil && pkg.Name == shortName {
			pLen := commonPrefixLength(parentPkgPath, pkgPath)
			if pLen > bestPrefix {
				bestPrefix = pLen
				bestPath = pkgPath
			}
		}
	}
	return bestPath
}
```

**Step 2: Update package_resolve_test.go**

Replace direct `globalPackageCache`/`globalCacheMutex` manipulation with `Cache().Seed()` or direct cache setup via the test helper. The tests in `package_resolve_test.go` directly set cache entries — update them to use `Cache().Reset()` and `Cache().Seed()`.

**Step 3: Run tests**

Run: `go test ./internal/model/ -v -count=1 -race`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/model/package_resolve.go internal/model/package_resolve_test.go
git commit -m "refactor: migrate package_resolve to use PackageCache singleton"
```

---

### Task 4: Migrate orchestrator to use PackageCache

**Files:**
- Modify: `internal/orchestrator/schema_builder.go`
- Modify: `internal/orchestrator/service.go`

**Step 1: Simplify schema_builder.go**

The `preWarmPackages` function should be simplified to a single pass. Remove iterative passes 2-4 and `batchLoadPackages` helper. The remaining function:

```go
func preWarmPackages(work []structRefWork, debug Debugger) error {
	seen := make(map[string]bool)
	var toLoad []string
	for _, w := range work {
		if seen[w.pkgPath] || model.Cache().IsCached(w.pkgPath) {
			continue
		}
		seen[w.pkgPath] = true
		toLoad = append(toLoad, w.pkgPath)
	}

	if len(toLoad) == 0 {
		if debug != nil {
			debug.Printf("Orchestrator: Pre-warm: all %d packages already cached", len(work))
		}
		return nil
	}

	if debug != nil {
		debug.Printf("Orchestrator: Pre-warm: batch loading %d packages", len(toLoad))
	}

	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedName | packages.NeedImports | packages.NeedDeps,
		Fset: token.NewFileSet(),
	}
	pkgs, err := packages.Load(cfg, toLoad...)
	if err != nil {
		return fmt.Errorf("pre-warm batch load: %w", err)
	}

	model.Cache().Seed(pkgs)
	model.SeedEnumPackageCache(pkgs)

	if debug != nil {
		debug.Printf("Orchestrator: Pre-warm: seeded %d packages", len(pkgs))
	}
	return nil
}
```

Remove the `batchLoadPackages` helper function (no longer needed separately).

**Step 2: Update service.go**

Replace `model.SeedGlobalPackageCache(loadResult.Packages)` with `model.Cache().Seed(loadResult.Packages)`.
Replace `model.GlobalCacheStats()` with `model.Cache().Stats()`.

(Or leave the wrapper functions from Task 2 and update these later — either way works.)

**Step 3: Update buildSchemasRecursive candidate search**

In `struct_field_lookup.go` `buildSchemasRecursive`, the candidate search at ~line 1063 directly iterates `globalPackageCache`. Update to use `Cache()`:

```go
// Replace:
//   globalCacheMutex.RLock()
//   for cachedPath, cachedPkg := range globalPackageCache {
// With:
Cache().RLock()
for cachedPath, cachedPkg := range Cache().Packages() {
```

And the corresponding `RUnlock`.

**Step 4: Run full test suite**

Run: `go test ./internal/... -v -count=1 -race`
Expected: All PASS

**Step 5: Run integration tests**

Run: `make test-project-1` and `make test-project-2`
Expected: test-project-1 produces 54 definitions (deterministic). test-project-2 produces a consistent count across multiple runs.

**Step 6: Commit**

```bash
git add internal/orchestrator/schema_builder.go internal/orchestrator/service.go internal/model/struct_field_lookup.go
git commit -m "refactor: simplify orchestrator to use PackageCache singleton

Remove iterative pre-warm passes 2-4, batchLoadPackages helper.
Pre-warm is now a single-pass performance optimization.
Cache().GetOrLoad() handles any misses during concurrent builds."
```

---

### Task 5: Verify determinism and clean up

**Step 1: Run test-project-2 multiple times to verify determinism**

Run 5 times:
```bash
for i in 1 2 3 4 5; do make test-project-2 2>&1 | grep -i "definitions\|schemas"; done
```

Expected: Same definition count every run.

**Step 2: Run the full test suite with race detector**

Run: `go test ./... -race -count=1`
Expected: All PASS, no races

**Step 3: Clean up any remaining references to old globals**

Search for any remaining direct references to the old variables:
```bash
grep -rn "globalPackageCache\|globalCacheMutex\|fullyLoaded\|packageLoadGroup" internal/
```

Expected: Zero results (all migrated to PackageCache).

**Step 4: Update change log**

Add entry to `.agents/change_log.md` documenting the fix.

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore: verify determinism and clean up old cache references"
```
