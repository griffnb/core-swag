package model

import (
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

// resolvePackagePath resolves a short package name (e.g., "constants") to its
// full import path, given the parent package path for context. It first tries
// sibling-replace (replacing the last path segment), then searches the global
// and enum package caches for a matching package name, preferring the one with
// the longest common path prefix with parentPkgPath.
func resolvePackagePath(parentPkgPath, shortName string) string {
	// Build the sibling guess (replace last segment)
	siblingPath := shortName
	if idx := strings.LastIndex(parentPkgPath, "/"); idx >= 0 {
		siblingPath = parentPkgPath[:idx+1] + shortName
	}

	// Fast path: sibling exists in the package cache
	if Cache().get(siblingPath) != nil {
		return siblingPath
	}

	// Sibling not in cache — search the PackageCache for packages whose
	// .Name matches shortName, then pick the one closest to parentPkgPath.
	best := searchPackageCacheByName(parentPkgPath, shortName)
	if best != "" {
		return best
	}

	// Also search the enum package cache
	best = searchCacheByName(enumPackageCache, &enumCacheMutex, parentPkgPath, shortName)
	if best != "" {
		return best
	}

	// Graceful degradation: return the sibling guess
	return siblingPath
}

// searchPackageCacheByName searches the PackageCache singleton for a package
// whose .Name matches shortName. When multiple matches exist it returns the
// one with the longest common path prefix with parentPkgPath.
func searchPackageCacheByName(parentPkgPath, shortName string) string {
	Cache().RLock()
	defer Cache().RUnlock()

	var bestPath string
	bestPrefix := -1
	for pkgPath, pkg := range Cache().Packages() {
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

// searchCacheByName iterates a package cache looking for packages whose .Name
// matches shortName. When multiple matches exist it returns the one with the
// longest common path prefix with parentPkgPath.
func searchCacheByName(cache map[string]*packages.Package, mu *sync.RWMutex, parentPkgPath, shortName string) string {
	mu.RLock()
	defer mu.RUnlock()

	var bestPath string
	bestPrefix := -1
	for pkgPath, pkg := range cache {
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

// commonPrefixLength returns the length of the longest common prefix between
// two strings, measured in bytes.
func commonPrefixLength(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
