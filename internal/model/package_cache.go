package model

import (
	"fmt"
	"go/token"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/griffnb/core-swag/internal/console"
	"golang.org/x/sync/singleflight"
	"golang.org/x/tools/go/packages"
)

// PackageCache is a concurrency-safe cache for *packages.Package values. It
// deduplicates concurrent loads via singleflight and tracks hit/miss stats.
// All exported methods are safe for concurrent use.
type PackageCache struct {
	mu        sync.RWMutex
	packages  map[string]*packages.Package
	loadGroup singleflight.Group
	hits      int64
	misses    int64
}

// singleton holds the process-wide PackageCache instance.
var (
	cacheOnce     sync.Once
	cacheInstance *PackageCache
)

// Cache returns the process-wide PackageCache singleton.
func Cache() *PackageCache {
	cacheOnce.Do(func() {
		cacheInstance = &PackageCache{
			packages: make(map[string]*packages.Package),
		}
	})
	return cacheInstance
}

// GetOrLoad returns the cached *packages.Package for pkgPath if it has Syntax.
// Otherwise it loads the package (and its transitive deps) via packages.Load,
// caches everything, and returns the result. Concurrent calls for the same
// pkgPath are deduplicated by singleflight. Returns nil if the package cannot
// be loaded.
func (pc *PackageCache) GetOrLoad(pkgPath string) *packages.Package {
	pc.mu.RLock()
	pkg, ok := pc.packages[pkgPath]
	pc.mu.RUnlock()

	if ok && len(pkg.Syntax) > 0 {
		atomic.AddInt64(&pc.hits, 1)
		return pkg
	}

	atomic.AddInt64(&pc.misses, 1)

	val, err, _ := pc.loadGroup.Do(pkgPath, func() (any, error) {
		cfg := &packages.Config{
			Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
				packages.NeedName | packages.NeedImports | packages.NeedDeps,
			Fset: token.NewFileSet(),
		}

		loadPath := pkgPath
		// Relative module paths (e.g., "design/models/icon_tag") don't have a
		// dot in the first segment so packages.Load treats them as stdlib and
		// fails. Prepend "./" so the Go toolchain resolves them relative to the
		// working directory instead.
		if !strings.Contains(pkgPath, ".") && !strings.HasPrefix(pkgPath, "./") {
			loadPath = "./" + pkgPath
		}

		pkgs, loadErr := packages.Load(cfg, loadPath)
		if loadErr != nil || len(pkgs) == 0 {
			return nil, fmt.Errorf("failed to load package %s: %v", pkgPath, loadErr)
		}

		result := pkgs[0]

		pc.mu.Lock()
		pc.seedLocked(pkgs)
		pc.mu.Unlock()

		console.Logger.Debug("PackageCache: cached package tree from %s\n", pkgPath)
		return result, nil
	})
	if err != nil {
		// Reload failed. If we had a cached entry (even without Syntax),
		// return it — its Types info is still useful for nested type resolution.
		// This prevents non-deterministic nil returns when transitive deps
		// can't be individually reloaded (e.g., relative module paths).
		if ok {
			console.Logger.Debug("PackageCache: LOAD FAILED for %s, returning cached entry (no Syntax): %v\n", pkgPath, err)
			return pkg
		}
		console.Logger.Debug("PackageCache: LOAD FAILED for %s: %v\n", pkgPath, err)
		return nil
	}

	return val.(*packages.Package)
}

// Seed bulk-populates the cache from a batch of packages. Direct packages
// (the slice elements themselves) always overwrite existing entries. Transitive
// deps discovered by walking Imports are only cached if no entry exists yet.
// Safe to call with nil, empty slices, or slices containing nil elements.
func (pc *PackageCache) Seed(pkgs []*packages.Package) {
	if len(pkgs) == 0 {
		return
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.seedLocked(pkgs)
}

// seedLocked is the lock-free Seed implementation. The caller must hold pc.mu
// (write lock). It is also called by the GetOrLoad singleflight callback.
func (pc *PackageCache) seedLocked(pkgs []*packages.Package) {
	visited := make(map[string]bool)

	// Pass 1: direct packages overwrite unconditionally. These come from the
	// initial packages.Load call and have full Syntax/TypesInfo.
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		pc.packages[pkg.PkgPath] = pkg
		visited[pkg.PkgPath] = true
	}

	// Pass 2: walk transitive imports. Only cache packages that have no
	// existing entry (preserves syntax-bearing entries from earlier loads).
	var walk func(pkg *packages.Package)
	walk = func(pkg *packages.Package) {
		if pkg == nil {
			return
		}
		if visited[pkg.PkgPath] {
			return
		}
		visited[pkg.PkgPath] = true

		existing := pc.packages[pkg.PkgPath]
		if existing == nil || (len(existing.Syntax) == 0 && len(pkg.Syntax) > 0) {
			pc.packages[pkg.PkgPath] = pkg
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

// IsCached reports whether pkgPath is in the cache AND has AST Syntax. A
// package without Syntax is not considered usable for struct field extraction.
func (pc *PackageCache) IsCached(pkgPath string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	pkg, ok := pc.packages[pkgPath]
	return ok && len(pkg.Syntax) > 0
}

// Stats returns the current cache hit and miss counts.
func (pc *PackageCache) Stats() (hits, misses int64) {
	return atomic.LoadInt64(&pc.hits), atomic.LoadInt64(&pc.misses)
}

// ResetStats zeroes the hit/miss counters. Intended for test isolation.
func (pc *PackageCache) ResetStats() {
	atomic.StoreInt64(&pc.hits, 0)
	atomic.StoreInt64(&pc.misses, 0)
}

// Reset clears the entire cache and resets stats. Intended for test isolation.
func (pc *PackageCache) Reset() {
	pc.mu.Lock()
	pc.packages = make(map[string]*packages.Package)
	pc.mu.Unlock()

	pc.ResetStats()
}

// get is a simple cache lookup. Returns nil if pkgPath is not cached.
func (pc *PackageCache) get(pkgPath string) *packages.Package {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.packages[pkgPath]
}

// Packages returns the internal map reference. Callers must hold at least a
// read lock (via RLock) and must not modify the returned map.
func (pc *PackageCache) Packages() map[string]*packages.Package {
	return pc.packages
}

// RLock acquires a read lock on the cache. Callers iterating over Packages()
// must hold this lock for the duration of their iteration.
func (pc *PackageCache) RLock() {
	pc.mu.RLock()
}

// RUnlock releases the read lock acquired by RLock.
func (pc *PackageCache) RUnlock() {
	pc.mu.RUnlock()
}
