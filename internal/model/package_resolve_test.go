package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"
)

func TestResolvePackagePath_SiblingFoundInCache(t *testing.T) {
	resetGlobalPackageCache()
	resetEnumPackageCache()

	// Seed the global cache with a sibling package
	globalCacheMutex.Lock()
	globalPackageCache["github.com/org/proj/internal/models/constants"] = &packages.Package{
		PkgPath: "github.com/org/proj/internal/models/constants",
		Name:    "constants",
	}
	globalCacheMutex.Unlock()

	result := resolvePackagePath("github.com/org/proj/internal/models/account", "constants")
	assert.Equal(t, "github.com/org/proj/internal/models/constants", result)
}

func TestResolvePackagePath_SiblingNotInCache_RealPathFound(t *testing.T) {
	resetGlobalPackageCache()
	resetEnumPackageCache()

	// The sibling path would be .../models/constants but the real package
	// is at .../internal/constants (not a sibling)
	globalCacheMutex.Lock()
	globalPackageCache["github.com/org/proj/internal/constants"] = &packages.Package{
		PkgPath: "github.com/org/proj/internal/constants",
		Name:    "constants",
	}
	globalCacheMutex.Unlock()

	result := resolvePackagePath("github.com/org/proj/internal/models/account", "constants")
	assert.Equal(t, "github.com/org/proj/internal/constants", result)
}

func TestResolvePackagePath_MultipleMatches_PrefersLongestPrefix(t *testing.T) {
	resetGlobalPackageCache()
	resetEnumPackageCache()

	// Two packages named "constants" at different locations
	globalCacheMutex.Lock()
	globalPackageCache["github.com/org/proj/internal/constants"] = &packages.Package{
		PkgPath: "github.com/org/proj/internal/constants",
		Name:    "constants",
	}
	globalPackageCache["github.com/other/lib/constants"] = &packages.Package{
		PkgPath: "github.com/other/lib/constants",
		Name:    "constants",
	}
	globalCacheMutex.Unlock()

	result := resolvePackagePath("github.com/org/proj/internal/models/account", "constants")
	// Should prefer the one in the same project (longer common prefix)
	assert.Equal(t, "github.com/org/proj/internal/constants", result)
}

func TestResolvePackagePath_NoMatch_ReturnsSiblingPath(t *testing.T) {
	resetGlobalPackageCache()
	resetEnumPackageCache()

	// Neither cache has a package named "constants"
	result := resolvePackagePath("github.com/org/proj/internal/models/account", "constants")
	assert.Equal(t, "github.com/org/proj/internal/models/constants", result)
}

func TestResolvePackagePath_FoundInEnumCache(t *testing.T) {
	resetGlobalPackageCache()
	resetEnumPackageCache()

	// Package only exists in enum cache, not global
	enumCacheMutex.Lock()
	enumPackageCache["github.com/org/proj/internal/constants"] = &packages.Package{
		PkgPath: "github.com/org/proj/internal/constants",
		Name:    "constants",
	}
	enumCacheMutex.Unlock()

	result := resolvePackagePath("github.com/org/proj/internal/models/account", "constants")
	assert.Equal(t, "github.com/org/proj/internal/constants", result)
}

func TestResolvePackagePath_NoSlashInParent(t *testing.T) {
	resetGlobalPackageCache()
	resetEnumPackageCache()

	// Edge case: parent path has no slash
	result := resolvePackagePath("main", "constants")
	assert.Equal(t, "constants", result)
}

func TestCommonPrefixLength(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"identical", "abc", "abc", 3},
		{"empty_a", "", "abc", 0},
		{"empty_b", "abc", "", 0},
		{"both_empty", "", "", 0},
		{"partial", "github.com/org/proj/internal", "github.com/org/proj/other", 20},
		{"no_common", "foo", "bar", 0},
		{"prefix_of_other", "abc", "abcdef", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, commonPrefixLength(tt.a, tt.b))
		})
	}
}
