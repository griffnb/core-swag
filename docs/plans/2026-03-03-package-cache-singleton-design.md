# PackageCache Singleton Design

## Problem

Package loading has two code paths: `LookupStructFields` can load packages on-demand, but `checkNamed`/`resolvePackage` is cache-only. When nested type resolution encounters a package not in the cache, it silently drops the field instead of loading the package. This causes non-deterministic schema counts depending on which packages happen to be pre-loaded.

## Solution

Replace all bare global cache variables with a `PackageCache` singleton struct that provides one method: `GetOrLoad(pkgPath) *packages.Package`. Every caller uses this single function. No silent nil returns — if a package isn't cached, it gets loaded.

## Design

### PackageCache struct

```go
type PackageCache struct {
    mu        sync.RWMutex
    packages  map[string]*packages.Package
    loadGroup singleflight.Group
    hits      int64
    misses    int64
}
```

### Singleton accessor

```go
var (
    cacheOnce     sync.Once
    cacheInstance *PackageCache
)

func Cache() *PackageCache {
    cacheOnce.Do(func() {
        cacheInstance = &PackageCache{
            packages: make(map[string]*packages.Package),
        }
    })
    return cacheInstance
}
```

### Core method

```go
func (c *PackageCache) GetOrLoad(pkgPath string) *packages.Package
```

1. Check cache — return if cached AND `len(pkg.Syntax) > 0`
2. Load via singleflight (deduplicates concurrent loads for same path)
3. Cache direct target + transitive deps
4. Return the package (or nil if genuinely unloadable)

### Supporting methods

- `Seed(pkgs []*packages.Package)` — bulk-populate from batch load results
- `IsCached(pkgPath string) bool` — check if usable entry exists (for pre-warm skip)
- `Stats() (hits, misses int64)` — cache statistics
- `ResetStats()` — zero counters for test isolation
- `Reset()` — clear entire cache for test isolation

### Cache usability check

A cached package is usable when `len(pkg.Syntax) > 0`. This is the only criterion — no `fullyLoaded` map, no distinguishing between "direct" and "transitive" loads.

## What changes

### Removed
- `fullyLoaded` map and all related functions (`NotFullyLoadedPaths`, `IsPackageCachedWithSyntax` replaced by `IsCached`)
- `forceReloadPackage()` — retry logic unnecessary with GetOrLoad
- `hasASTStructFields()` — AST checking for retry, unnecessary
- `resolvePackage()` method on `CoreStructParser` — replaced by `Cache().GetOrLoad()`
- Iterative pre-warm passes 2-4 — GetOrLoad handles misses
- Singleflight block inside `LookupStructFields` — replaced by `Cache().GetOrLoad()`
- Post-extraction retry block in `LookupStructFields`
- `len(builder.Fields) > 0` guard on shared cache writes
- Bare global vars: `globalPackageCache`, `globalCacheMutex`, `packageLoadGroup`

### Kept
- `SharedTypeCache` — different concern (cross-goroutine type result sharing)
- Pre-warm pass 1 in `preWarmPackages()` — performance optimization, simplified to single pass calling `Cache().Seed()`
- `CoreStructParser` — still does field extraction, calls `Cache().GetOrLoad()` instead of `resolvePackage()`
- `SeedEnumPackageCache` — separate cache for enum lookup (different concern)

### Callers updated
- `checkNamed`: `resolvePackage(pkg.Path())` -> `Cache().GetOrLoad(pkg.Path())`
- `LookupStructFields`: 80-line cache+singleflight+retry -> `Cache().GetOrLoad(importPath)`
- `processStructField`: `resolvePackage(subTypePackage)` -> `Cache().GetOrLoad(subTypePackage)`
- `SeedGlobalPackageCache` callers -> `Cache().Seed()`
- `preWarmPackages`: simplified to single pass + `Cache().Seed()`
- All test helpers that reset global state -> `Cache().Reset()`
