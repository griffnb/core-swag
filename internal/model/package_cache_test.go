package model

import (
	"go/ast"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// newTestCache creates a fresh PackageCache for test isolation. It does NOT
// use the singleton so parallel tests never interfere with each other.
func newTestCache() *PackageCache {
	return &PackageCache{
		packages: make(map[string]*packages.Package),
	}
}

func TestCache_ReturnsSingleton(t *testing.T) {
	a := Cache()
	b := Cache()
	assert.Same(t, a, b, "Cache() must return the same pointer on every call")
}

func TestPackageCache_Seed_AddsEntries(t *testing.T) {
	pc := newTestCache()

	pkgA := &packages.Package{PkgPath: "example.com/a", Name: "a"}
	pkgB := &packages.Package{PkgPath: "example.com/b", Name: "b"}

	pc.Seed([]*packages.Package{pkgA, pkgB})

	assert.Equal(t, pkgA, pc.get("example.com/a"))
	assert.Equal(t, pkgB, pc.get("example.com/b"))
}

func TestPackageCache_Seed_WalksTransitiveImports(t *testing.T) {
	pc := newTestCache()

	leaf := &packages.Package{
		PkgPath: "example.com/leaf",
		Imports: map[string]*packages.Package{},
	}
	root := &packages.Package{
		PkgPath: "example.com/root",
		Imports: map[string]*packages.Package{
			"example.com/leaf": leaf,
		},
	}

	pc.Seed([]*packages.Package{root})

	assert.Equal(t, root, pc.get("example.com/root"), "root should be cached")
	assert.Equal(t, leaf, pc.get("example.com/leaf"), "transitive dep should be cached")
}

func TestPackageCache_Seed_HandlesCircularImports(t *testing.T) {
	pc := newTestCache()

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

	assert.NotPanics(t, func() {
		pc.Seed([]*packages.Package{pkgA})
	})

	require.Equal(t, pkgA, pc.get("example.com/a"))
	require.Equal(t, pkgB, pc.get("example.com/b"))
}

func TestPackageCache_Seed_NilAndEmpty(t *testing.T) {
	pc := newTestCache()

	assert.NotPanics(t, func() { pc.Seed(nil) }, "nil slice must not panic")
	assert.NotPanics(t, func() { pc.Seed([]*packages.Package{}) }, "empty slice must not panic")
	assert.NotPanics(t, func() { pc.Seed([]*packages.Package{nil, nil}) }, "slice with nils must not panic")

	// Cache should remain empty after all of the above.
	pc.RLock()
	defer pc.RUnlock()
	assert.Empty(t, pc.Packages())
}

func TestPackageCache_Seed_DirectOverwritesTransitive(t *testing.T) {
	pc := newTestCache()

	// First, seed a transitive dep (no Syntax).
	transitive := &packages.Package{PkgPath: "example.com/dep", Name: "transitive"}
	root := &packages.Package{
		PkgPath: "example.com/root",
		Imports: map[string]*packages.Package{
			"example.com/dep": transitive,
		},
	}
	pc.Seed([]*packages.Package{root})
	assert.Equal(t, "transitive", pc.get("example.com/dep").Name)

	// Now seed with the dep as a direct package (should overwrite).
	direct := &packages.Package{PkgPath: "example.com/dep", Name: "direct"}
	pc.Seed([]*packages.Package{direct})
	assert.Equal(t, "direct", pc.get("example.com/dep").Name,
		"direct packages must overwrite previously cached transitive deps")
}

func TestPackageCache_IsCached_WithSyntax(t *testing.T) {
	pc := newTestCache()

	withSyntax := &packages.Package{
		PkgPath: "example.com/withsyntax",
		Syntax:  []*ast.File{{}},
	}
	withoutSyntax := &packages.Package{
		PkgPath: "example.com/withoutsyntax",
	}
	pc.Seed([]*packages.Package{withSyntax, withoutSyntax})

	assert.True(t, pc.IsCached("example.com/withsyntax"),
		"package with Syntax should report as cached")
	assert.False(t, pc.IsCached("example.com/withoutsyntax"),
		"package without Syntax should report as not cached")
	assert.False(t, pc.IsCached("example.com/nonexistent"),
		"nonexistent package should report as not cached")
}

func TestPackageCache_Stats(t *testing.T) {
	pc := newTestCache()

	hits, misses := pc.Stats()
	assert.Equal(t, int64(0), hits, "fresh cache should have zero hits")
	assert.Equal(t, int64(0), misses, "fresh cache should have zero misses")
}

func TestPackageCache_Get_ReturnsNilForMissing(t *testing.T) {
	pc := newTestCache()

	assert.Nil(t, pc.get("example.com/nonexistent"),
		"get() on empty cache must return nil")
}

func TestPackageCache_ConcurrentSeed(t *testing.T) {
	pc := newTestCache()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			pkg := &packages.Package{
				PkgPath: "example.com/concurrent",
				Name:    "concurrent",
			}
			pc.Seed([]*packages.Package{pkg})
		}(i)
	}
	wg.Wait()

	// If the race detector doesn't fire and we reach here, the test passes.
	assert.NotNil(t, pc.get("example.com/concurrent"),
		"package should be present after concurrent seeds")
}

func TestPackageCache_ResetStats(t *testing.T) {
	pc := newTestCache()

	// Manually bump counters to verify reset clears them.
	pc.hits = 5
	pc.misses = 3

	pc.ResetStats()

	hits, misses := pc.Stats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)
}

func TestPackageCache_Reset(t *testing.T) {
	pc := newTestCache()

	pkg := &packages.Package{PkgPath: "example.com/a"}
	pc.Seed([]*packages.Package{pkg})
	pc.hits = 10
	pc.misses = 5

	pc.Reset()

	assert.Nil(t, pc.get("example.com/a"), "cache should be empty after Reset")
	hits, misses := pc.Stats()
	assert.Equal(t, int64(0), hits, "hits should be zero after Reset")
	assert.Equal(t, int64(0), misses, "misses should be zero after Reset")
}

func TestPackageCache_Packages_ReturnsSameMap(t *testing.T) {
	pc := newTestCache()

	pkg := &packages.Package{PkgPath: "example.com/a"}
	pc.Seed([]*packages.Package{pkg})

	pc.RLock()
	m := pc.Packages()
	pc.RUnlock()

	assert.Len(t, m, 1)
	assert.Equal(t, pkg, m["example.com/a"])
}

func TestPackageCache_RLockRUnlock(t *testing.T) {
	pc := newTestCache()

	// Verify RLock/RUnlock don't deadlock and allow concurrent reads.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pc.RLock()
			_ = pc.Packages()
			pc.RUnlock()
		}()
	}
	wg.Wait()
}
