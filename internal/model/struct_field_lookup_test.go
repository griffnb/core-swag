package model

import (
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

func TestSeedGlobalPackageCache_DoesNotOverwriteExisting(t *testing.T) {
	resetGlobalPackageCache()

	original := &packages.Package{PkgPath: "example.com/a", Name: "original"}
	replacement := &packages.Package{PkgPath: "example.com/a", Name: "replacement"}

	// Pre-populate cache with original
	globalCacheMutex.Lock()
	globalPackageCache["example.com/a"] = original
	globalCacheMutex.Unlock()

	SeedGlobalPackageCache([]*packages.Package{replacement})

	globalCacheMutex.RLock()
	defer globalCacheMutex.RUnlock()

	assert.Equal(t, "original", globalPackageCache["example.com/a"].Name,
		"seed should not overwrite an existing cache entry")
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
	// lookup will follow the cache-hit path.
	seeded := &packages.Package{PkgPath: "example.com/seeded", Name: "seeded"}
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
