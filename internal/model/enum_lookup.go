package model

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"sync"

	"github.com/griffnb/core-swag/internal/console"
	"golang.org/x/tools/go/packages"
)

// Global package cache for enum lookup
var (
	enumPackageCache = make(map[string]*packages.Package)
	enumCacheMutex   sync.RWMutex
)

// SeedEnumPackageCache pre-populates the enum package cache with all packages
// and their transitive imports. This avoids redundant packages.Load calls
// during enum resolution by warming the cache upfront.
func SeedEnumPackageCache(pkgs []*packages.Package) {
	if len(pkgs) == 0 {
		return
	}

	visited := make(map[string]bool)

	var walk func(pkg *packages.Package)
	walk = func(pkg *packages.Package) {
		if pkg == nil {
			return
		}
		if visited[pkg.PkgPath] {
			return
		}
		visited[pkg.PkgPath] = true

		if enumPackageCache[pkg.PkgPath] == nil {
			enumPackageCache[pkg.PkgPath] = pkg
		}

		for _, imp := range pkg.Imports {
			walk(imp)
		}
	}

	enumCacheMutex.Lock()
	defer enumCacheMutex.Unlock()

	for _, pkg := range pkgs {
		walk(pkg)
	}
}

// ParserEnumLookup implements TypeEnumLookup using CoreStructParser
type ParserEnumLookup struct {
	Parser        *CoreStructParser
	BaseModule    string
	PkgPath       string
	packageCache  map[string]*packages.Package // Local cache for loaded packages
	cacheMutex    sync.RWMutex                 // Protect local cache
	sharedFileSet *token.FileSet               // Shared FileSet for fallback packages.Load calls
}

// getOrCreateFileSet returns the shared FileSet, creating one if needed.
// token.FileSet is internally thread-safe so no additional mutex is required.
func (p *ParserEnumLookup) getOrCreateFileSet() *token.FileSet {
	if p.sharedFileSet == nil {
		p.sharedFileSet = token.NewFileSet()
	}
	return p.sharedFileSet
}

// GetEnumsForType looks up enum values for a given type name
// typeName should be fully qualified like "constants.Role" or just "Role"
// or a full package path like "github.com/griffnb/core-swag/testing/testdata/core_models/constants.Role"
func (p *ParserEnumLookup) GetEnumsForType(typeName string, file *ast.File) ([]EnumValue, error) {
	console.Logger.Debug("$Bold{$Cyan{>>> GetEnumsForType called with typeName: %s}}\n", typeName)
	if p.Parser == nil {
		console.Logger.Debug("$Bold{$Red{>>> GetEnumsForType: parser is nil!}}\n")
		return nil, fmt.Errorf("parser is nil")
	}

	// Parse the type name to extract package path and type name
	// Handle both "constants.Role" and "github.com/.../constants.Role"
	var targetPkgPath, baseTypeName string
	lastDot := strings.LastIndex(typeName, ".")
	if lastDot == -1 {
		// No dot, just a type name - use the current package path
		baseTypeName = typeName
		targetPkgPath = p.PkgPath
	} else {
		baseTypeName = typeName[lastDot+1:]
		// Everything before the last dot could be:
		// 1. A short package name like "constants" (need to resolve to full path)
		// 2. A full package path like "github.com/.../constants"
		pkgPart := typeName[:lastDot]

		// Check if it's a full path (contains slash) or short name
		if strings.Contains(pkgPart, "/") {
			// Full path - use as-is
			targetPkgPath = pkgPart
		} else {
			// Short package name - resolve relative to current package path
			// e.g., if we're in "github.com/.../account" and see "constants",
			// resolve to "github.com/.../constants"
			if p.PkgPath != "" && strings.Contains(p.PkgPath, "/") {
				lastSlash := strings.LastIndex(p.PkgPath, "/")
				targetPkgPath = p.PkgPath[:lastSlash+1] + pkgPart
			} else {
				// Fallback: use short name as-is (might fail)
				targetPkgPath = pkgPart
			}
		}
	}

	console.Logger.Debug("$Bold{$Red{Looking up enums for type: %s in package: %s (path: %s)}}\n", baseTypeName, "", targetPkgPath)

	// Initialize local cache if needed
	p.cacheMutex.Lock()
	if p.packageCache == nil {
		p.packageCache = make(map[string]*packages.Package)
	}
	p.cacheMutex.Unlock()

	// Check global cache first
	enumCacheMutex.RLock()
	pkg, pkgCached := enumPackageCache[targetPkgPath]
	enumCacheMutex.RUnlock()

	if !pkgCached {
		// Check local cache
		p.cacheMutex.RLock()
		pkg, pkgCached = p.packageCache[targetPkgPath]
		p.cacheMutex.RUnlock()
	}

	if !pkgCached {
		// Load the package
		cfg := &packages.Config{
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
				packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
			Fset: p.getOrCreateFileSet(),
		}

		pkgs, err := packages.Load(cfg, targetPkgPath)
		if err != nil {
			return nil, err
		}

		if len(pkgs) == 0 {
			return nil, fmt.Errorf("no packages found for %s", targetPkgPath)
		}

		pkg = pkgs[0]

		// Store in both caches
		enumCacheMutex.Lock()
		enumPackageCache[targetPkgPath] = pkg
		enumCacheMutex.Unlock()

		p.cacheMutex.Lock()
		p.packageCache[targetPkgPath] = pkg
		p.cacheMutex.Unlock()
	}

	// Look for the type definition and collect const values
	var enums []EnumValue
	var typeFound bool

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			// First, find the type definition
			if genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok && typeSpec.Name.Name == baseTypeName {
						typeFound = true
						break
					}
				}
			}

			// Collect constants of this type
			if genDecl.Tok == token.CONST && typeFound {
				for _, spec := range genDecl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}

					// Check each name in the value spec
					for _, name := range valueSpec.Names {
						// Use TypesInfo to determine the actual type of the constant
						// This handles both explicit types and iota expressions
						if pkg.TypesInfo != nil {
							if constObj, ok := pkg.TypesInfo.Defs[name].(*types.Const); ok {
								// Check if this constant is of the target type
								typeName := constObj.Type().String()
								// Extract the base type name from the full type path
								lastDotIdx := strings.LastIndex(typeName, ".")
								var constTypeName string
								if lastDotIdx != -1 {
									constTypeName = typeName[lastDotIdx+1:]
								} else {
									constTypeName = typeName
								}

								if constTypeName == baseTypeName {
									value := constObj.Val()
									comment := ""
									if valueSpec.Comment != nil && len(valueSpec.Comment.List) > 0 {
										comment = strings.TrimSpace(strings.TrimPrefix(valueSpec.Comment.List[0].Text, "//"))
									} else if valueSpec.Doc != nil && len(valueSpec.Doc.List) > 0 {
										comment = strings.TrimSpace(strings.TrimPrefix(valueSpec.Doc.List[len(valueSpec.Doc.List)-1].Text, "//"))
									}

									// Convert constant value to the appropriate Go type
									var enumValue interface{}
									switch value.Kind() {
									case constant.Int:
										// Convert to int64 then to int
										if v, ok := constant.Int64Val(value); ok {
											enumValue = int(v)
										}
									case constant.String:
										// ExactString includes quotes, so use StringVal
										enumValue = constant.StringVal(value)
									default:
										// Fallback to string representation
										enumValue = value.ExactString()
									}

									enums = append(enums, EnumValue{
										Key:     name.Name,
										Value:   enumValue,
										Comment: comment,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	if !typeFound {
		return nil, fmt.Errorf("type %s not found", baseTypeName)
	}

	dedupedEnums := make([]EnumValue, 0, len(enums))
	seenValues := make(map[interface{}]bool)
	for _, enum := range enums {
		if _, seen := seenValues[enum.Value]; !seen {
			seenValues[enum.Value] = true
			dedupedEnums = append(dedupedEnums, enum)
		}
	}

	return dedupedEnums, nil
}
