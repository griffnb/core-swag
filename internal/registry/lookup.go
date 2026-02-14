// Package registry - type lookup and import resolution functionality.
package registry

import (
	"go/ast"
	"strings"

	"github.com/swaggo/swag/internal/domain"
)

// findPackagePathFromImports finds the package path from imports.
func (s *Service) findPackagePathFromImports(pkg string, file *ast.File) (matchedPkgPaths, externalPkgPaths []string) {
	if file == nil {
		return
	}

	if strings.ContainsRune(pkg, '.') {
		pkg = strings.Split(pkg, ".")[0]
	}

	matchLastPathPart := func(pkgPath string) bool {
		paths := strings.Split(pkgPath, "/")
		return paths[len(paths)-1] == pkg
	}

	// prior to match named package
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			if imp.Name.Name == pkg {
				// if name match, break loop and return
				_, ok := s.packages[path]
				if ok {
					matchedPkgPaths = []string{path}
					externalPkgPaths = nil
				} else {
					externalPkgPaths = []string{path}
					matchedPkgPaths = nil
				}
				break
			} else if imp.Name.Name == "_" && len(pkg) > 0 {
				// for unused types
				pd, ok := s.packages[path]
				if ok {
					if pd.Name == pkg {
						matchedPkgPaths = append(matchedPkgPaths, path)
					}
				} else if matchLastPathPart(path) {
					externalPkgPaths = append(externalPkgPaths, path)
				}
			} else if imp.Name.Name == "." && len(pkg) == 0 {
				_, ok := s.packages[path]
				if ok {
					matchedPkgPaths = append(matchedPkgPaths, path)
				} else if len(pkg) == 0 || matchLastPathPart(path) {
					externalPkgPaths = append(externalPkgPaths, path)
				}
			}
		} else if s.packages != nil && len(pkg) > 0 {
			pd, ok := s.packages[path]
			if ok {
				if pd.Name == pkg {
					matchedPkgPaths = append(matchedPkgPaths, path)
				}
			} else if matchLastPathPart(path) {
				externalPkgPaths = append(externalPkgPaths, path)
			}
		}
	}

	if len(pkg) == 0 || file.Name.Name == pkg {
		if fi, ok := s.files[file]; ok {
			matchedPkgPaths = append(matchedPkgPaths, fi.PackagePath)
		}
	}

	return
}

func (s *Service) findTypeSpecFromPackagePaths(matchedPkgPaths, externalPkgPaths []string, name string) (typeDef *domain.TypeSpecDef) {
	if s.parseDependency > 0 {
		for _, pkgPath := range externalPkgPaths {
			if err := s.loadExternalPackage(pkgPath); err == nil {
				typeDef = s.findTypeSpec(pkgPath, name)
				if typeDef != nil {
					return typeDef
				}
			}
		}
	}

	for _, pkgPath := range matchedPkgPaths {
		typeDef = s.findTypeSpec(pkgPath, name)
		if typeDef != nil {
			return typeDef
		}
	}

	return typeDef
}

func (s *Service) parametrizeGenericType(file *ast.File, typeDef *domain.TypeSpecDef, typeName string) *domain.TypeSpecDef {
	if typeDef == nil || typeDef.TypeSpec.TypeParams == nil {
		return typeDef
	}

	// This is a placeholder for generic type parametrization
	// The actual implementation is complex and involves type parameter substitution
	return typeDef
}
