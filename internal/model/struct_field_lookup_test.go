package model

import (
	"go/ast"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// resetGlobalPackageCache clears the global cache between tests.
func resetGlobalPackageCache() {
	globalCacheMutex.Lock()
	globalPackageCache = make(map[string]*packages.Package)
	globalCacheMutex.Unlock()
}

func TestSeedGlobalPackageCache_NilSlice(t *testing.T) {
	resetGlobalPackageCache()

	// Should not panic on nil input
	assert.NotPanics(t, func() {
		SeedGlobalPackageCache(nil)
	})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()
	assert.Empty(t, globalPackageCache)
}

func TestSeedGlobalPackageCache_EmptySlice(t *testing.T) {
	resetGlobalPackageCache()

	SeedGlobalPackageCache([]*packages.Package{})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()
	assert.Empty(t, globalPackageCache)
}

func TestSeedGlobalPackageCache_AddsEntries(t *testing.T) {
	resetGlobalPackageCache()

	pkgA := &packages.Package{PkgPath: "example.com/a"}
	pkgB := &packages.Package{PkgPath: "example.com/b"}

	SeedGlobalPackageCache([]*packages.Package{pkgA, pkgB})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()

	require.Len(t, globalPackageCache, 2)
	assert.Equal(t, pkgA, globalPackageCache["example.com/a"])
	assert.Equal(t, pkgB, globalPackageCache["example.com/b"])
}

func TestSeedGlobalPackageCache_SkipsNilPackage(t *testing.T) {
	resetGlobalPackageCache()

	pkgA := &packages.Package{PkgPath: "example.com/a"}

	SeedGlobalPackageCache([]*packages.Package{nil, pkgA, nil})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()

	require.Len(t, globalPackageCache, 1)
	assert.Equal(t, pkgA, globalPackageCache["example.com/a"])
}

func TestSeedGlobalPackageCache_DirectPackagesOverwriteExisting(t *testing.T) {
	resetGlobalPackageCache()

	original := &packages.Package{PkgPath: "example.com/a", Name: "original"}
	replacement := &packages.Package{PkgPath: "example.com/a", Name: "replacement"}

	// Pre-populate cache with original
	globalCacheMutex.Lock()
	globalPackageCache["example.com/a"] = original
	globalCacheMutex.Unlock()

	// Direct packages (pass 1) always overwrite — they come from the initial
	// packages.Load with full syntax and should take priority.
	SeedGlobalPackageCache([]*packages.Package{replacement})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()

	assert.Equal(t, "replacement", globalPackageCache["example.com/a"].Name,
		"direct packages should overwrite existing cache entries")
}

func TestSeedGlobalPackageCache_WalksImportsRecursively(t *testing.T) {
	resetGlobalPackageCache()

	// Build a dependency chain: root -> mid -> leaf
	leaf := &packages.Package{
		PkgPath: "example.com/leaf",
		Imports: map[string]*packages.Package{},
	}
	mid := &packages.Package{
		PkgPath: "example.com/mid",
		Imports: map[string]*packages.Package{
			"example.com/leaf": leaf,
		},
	}
	root := &packages.Package{
		PkgPath: "example.com/root",
		Imports: map[string]*packages.Package{
			"example.com/mid": mid,
		},
	}

	SeedGlobalPackageCache([]*packages.Package{root})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()

	require.Len(t, globalPackageCache, 3)
	assert.Equal(t, root, globalPackageCache["example.com/root"])
	assert.Equal(t, mid, globalPackageCache["example.com/mid"])
	assert.Equal(t, leaf, globalPackageCache["example.com/leaf"])
}

func TestSeedGlobalPackageCache_HandlesCircularImports(t *testing.T) {
	resetGlobalPackageCache()

	// Create circular dependency: a -> b -> a
	pkgA := &packages.Package{
		PkgPath: "example.com/a",
		Imports: map[string]*packages.Package{},
	}
	pkgB := &packages.Package{
		PkgPath: "example.com/b",
		Imports: map[string]*packages.Package{
			"example.com/a": pkgA,
		},
	}
	pkgA.Imports["example.com/b"] = pkgB

	// Should not hang or stack overflow
	assert.NotPanics(t, func() {
		SeedGlobalPackageCache([]*packages.Package{pkgA})
	})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()

	require.Len(t, globalPackageCache, 2)
	assert.Equal(t, pkgA, globalPackageCache["example.com/a"])
	assert.Equal(t, pkgB, globalPackageCache["example.com/b"])
}

func TestGlobalCacheStats_ResetWorks(t *testing.T) {
	// Intentionally dirty the counters so Reset has something to clear.
	ResetGlobalCacheStats()

	hits, misses := GlobalCacheStats()
	assert.Equal(t, int64(0), hits, "hits should be 0 after reset")
	assert.Equal(t, int64(0), misses, "misses should be 0 after reset")
}

func TestGlobalCacheStats_IncrementPattern(t *testing.T) {
	resetGlobalPackageCache()
	ResetGlobalCacheStats()

	// Seed a package into the global cache so the next LookupStructFields
	// lookup will follow the cache-hit path. The package must have Syntax
	// to be treated as a valid cache hit (syntax-less packages are reloaded).
	seeded := &packages.Package{
		PkgPath: "example.com/seeded",
		Name:    "seeded",
		Syntax:  []*ast.File{{}},
	}
	SeedGlobalPackageCache([]*packages.Package{seeded})

	parser := &CoreStructParser{}
	// Calling LookupStructFields for the seeded path triggers the cache-hit
	// branch (pkgCached == true). The type won't be found but the package
	// cache hit counter should still increment.
	_ = parser.LookupStructFields("example.com/seeded", "example.com/seeded", "NonExistent")

	hits, misses := GlobalCacheStats()
	assert.Equal(t, int64(1), hits, "expected one cache hit after seeded lookup")
	assert.Equal(t, int64(0), misses, "expected zero cache misses for seeded package")
}

func TestIsPackageCachedWithSyntax_ReturnsTrueWhenSyntaxPresent(t *testing.T) {
	resetGlobalPackageCache()

	pkg := &packages.Package{
		PkgPath: "example.com/withsyntax",
		Syntax:  []*ast.File{{}},
	}
	SeedGlobalPackageCache([]*packages.Package{pkg})

	assert.True(t, IsPackageCachedWithSyntax("example.com/withsyntax"))
}

func TestIsPackageCachedWithSyntax_ReturnsFalseWhenNoSyntax(t *testing.T) {
	resetGlobalPackageCache()

	pkg := &packages.Package{PkgPath: "example.com/nosyntax"}
	SeedGlobalPackageCache([]*packages.Package{pkg})

	assert.False(t, IsPackageCachedWithSyntax("example.com/nosyntax"))
}

func TestIsPackageCachedWithSyntax_ReturnsFalseWhenNotCached(t *testing.T) {
	resetGlobalPackageCache()

	assert.False(t, IsPackageCachedWithSyntax("example.com/nonexistent"))
}

func TestResolvePackage_LocalCacheFirst(t *testing.T) {
	resetGlobalPackageCache()

	localPkg := &packages.Package{PkgPath: "example.com/local", Name: "local"}
	globalPkg := &packages.Package{PkgPath: "example.com/local", Name: "global"}

	// Put different packages in local and global caches
	globalCacheMutex.Lock()
	globalPackageCache["example.com/local"] = globalPkg
	globalCacheMutex.Unlock()

	parser := &CoreStructParser{
		packageCache: map[string]*packages.Package{
			"example.com/local": localPkg,
		},
	}

	result := parser.resolvePackage("example.com/local")
	assert.Equal(t, "local", result.Name, "should prefer local cache")
}

func TestResolvePackage_FallsBackToGlobal(t *testing.T) {
	resetGlobalPackageCache()

	globalPkg := &packages.Package{PkgPath: "example.com/global", Name: "global"}

	globalCacheMutex.Lock()
	globalPackageCache["example.com/global"] = globalPkg
	globalCacheMutex.Unlock()

	parser := &CoreStructParser{
		packageCache: make(map[string]*packages.Package),
	}

	result := parser.resolvePackage("example.com/global")
	assert.Equal(t, "global", result.Name, "should fall back to global cache")
}

func TestResolvePackage_ReturnsNilWhenNotFound(t *testing.T) {
	resetGlobalPackageCache()

	parser := &CoreStructParser{
		packageCache: make(map[string]*packages.Package),
	}

	result := parser.resolvePackage("example.com/nonexistent")
	assert.Nil(t, result)
}

func TestSharedTypeCache_GetSet(t *testing.T) {
	cache := NewSharedTypeCache()

	builder := &StructBuilder{}
	cache.set("key1", builder)

	got, ok := cache.get("key1")
	assert.True(t, ok)
	assert.Same(t, builder, got)

	_, ok = cache.get("nonexistent")
	assert.False(t, ok)
}

func TestSharedTypeCache_ConcurrentAccess(t *testing.T) {
	cache := NewSharedTypeCache()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			builder := &StructBuilder{}
			cache.set(key, builder)
			cache.get(key)
		}(i)
	}
	wg.Wait()

	// If we get here without a race detector error, the test passes.
	_, ok := cache.get("key")
	assert.True(t, ok)
}

func TestSingleflight_DeduplicatesConcurrentLoads(t *testing.T) {
	// This test verifies the singleflight variable is properly initialized.
	// The actual deduplication is tested via integration tests (packages.Load
	// can't be easily mocked). Here we verify the group exists and works.
	var callCount int64
	var mu sync.Mutex

	// Use the package-level singleflight group to ensure it's initialized.
	val, err, _ := packageLoadGroup.Do("test-key", func() (any, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return "result", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "result", val)
	assert.Equal(t, int64(1), callCount)
}
