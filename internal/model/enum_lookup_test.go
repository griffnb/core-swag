package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// resetEnumPackageCache clears the enum cache between tests.
func resetEnumPackageCache() {
	enumCacheMutex.Lock()
	enumPackageCache = make(map[string]*packages.Package)
	enumCacheMutex.Unlock()
}

func TestSeedEnumPackageCache_NilSlice(t *testing.T) {
	resetEnumPackageCache()

	// Should not panic on nil input
	assert.NotPanics(t, func() {
		SeedEnumPackageCache(nil)
	})

	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()
	assert.Empty(t, enumPackageCache)
}

func TestSeedEnumPackageCache_EmptySlice(t *testing.T) {
	resetEnumPackageCache()

	SeedEnumPackageCache([]*packages.Package{})

	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()
	assert.Empty(t, enumPackageCache)
}

func TestSeedEnumPackageCache_AddsEntries(t *testing.T) {
	resetEnumPackageCache()

	pkgA := &packages.Package{PkgPath: "example.com/a"}
	pkgB := &packages.Package{PkgPath: "example.com/b"}

	SeedEnumPackageCache([]*packages.Package{pkgA, pkgB})

	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()

	require.Len(t, enumPackageCache, 2)
	assert.Equal(t, pkgA, enumPackageCache["example.com/a"])
	assert.Equal(t, pkgB, enumPackageCache["example.com/b"])
}

func TestSeedEnumPackageCache_SkipsNilPackage(t *testing.T) {
	resetEnumPackageCache()

	pkgA := &packages.Package{PkgPath: "example.com/a"}

	SeedEnumPackageCache([]*packages.Package{nil, pkgA, nil})

	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()

	require.Len(t, enumPackageCache, 1)
	assert.Equal(t, pkgA, enumPackageCache["example.com/a"])
}

func TestSeedEnumPackageCache_DoesNotOverwriteExisting(t *testing.T) {
	resetEnumPackageCache()

	original := &packages.Package{PkgPath: "example.com/a", Name: "original"}
	replacement := &packages.Package{PkgPath: "example.com/a", Name: "replacement"}

	// Pre-populate cache with original
	enumCacheMutex.Lock()
	enumPackageCache["example.com/a"] = original
	enumCacheMutex.Unlock()

	SeedEnumPackageCache([]*packages.Package{replacement})

	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()

	assert.Equal(t, "original", enumPackageCache["example.com/a"].Name,
		"seed should not overwrite an existing cache entry")
}

func TestSeedEnumPackageCache_WalksImportsRecursively(t *testing.T) {
	resetEnumPackageCache()

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

	SeedEnumPackageCache([]*packages.Package{root})

	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()

	require.Len(t, enumPackageCache, 3)
	assert.Equal(t, root, enumPackageCache["example.com/root"])
	assert.Equal(t, mid, enumPackageCache["example.com/mid"])
	assert.Equal(t, leaf, enumPackageCache["example.com/leaf"])
}

func TestSeedEnumPackageCache_HandlesCircularImports(t *testing.T) {
	resetEnumPackageCache()

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
		SeedEnumPackageCache([]*packages.Package{pkgA})
	})

	enumCacheMutex.RLock()
	defer enumCacheMutex.RUnlock()

	require.Len(t, enumPackageCache, 2)
	assert.Equal(t, pkgA, enumPackageCache["example.com/a"])
	assert.Equal(t, pkgB, enumPackageCache["example.com/b"])
}
