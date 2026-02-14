package loader

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// LoadWithGoPackages loads packages using go/packages
func (s *Service) LoadWithGoPackages(searchDirs []string, absMainAPIFilePath string) (*LoadResult, error) {
	mode := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
		packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo
	if s.parseDependency > 0 {
		mode |= packages.NeedDeps
	}

	absDirs := make([]string, 0, len(searchDirs)+1)
	absDirs = append(absDirs, filepath.Dir(absMainAPIFilePath))
	for _, dir := range searchDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		absDirs = append(absDirs, absDir+"/...")
	}

	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Mode: mode,
		Fset: fset,
	}, absDirs...)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			return nil, e
		}
	}

	result := &LoadResult{
		Files:    make(map[*ast.File]*AstFileInfo),
		Packages: pkgs,
	}

	err = s.walkPackages(pkgs, fset, result, pkgs)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// walkPackages walks packages loaded with go/packages
func (s *Service) walkPackages(pkgs []*packages.Package, fset *token.FileSet, result *LoadResult, rootPkgs []*packages.Package) error {
	pkgSeen := make(map[string]struct{})
	return s.walkPackagesInternal(pkgs, fset, result, rootPkgs, pkgSeen)
}

func (s *Service) walkPackagesInternal(pkgs []*packages.Package, fset *token.FileSet, result *LoadResult, rootPkgs []*packages.Package, pkgSeen map[string]struct{}) error {
	for _, pkg := range pkgs {
		if s.skipPackageByPrefix(pkg.PkgPath) {
			continue
		}
		if _, ok := pkgSeen[pkg.PkgPath]; ok {
			continue
		}
		pkgSeen[pkg.PkgPath] = struct{}{}

		parseFlag := ParseFlag(ParseAll)
		if !contains(rootPkgs, pkg) {
			parseFlag = ParseFlag(s.parseDependency)
		}

		for i, file := range pkg.CompiledGoFiles {
			fileInfo, err := os.Stat(file)
			if err != nil {
				return err
			}
			if s.shouldSkipDir(file, fileInfo) != nil {
				continue
			}

			info := &AstFileInfo{
				File:        pkg.Syntax[i],
				Path:        file,
				PackagePath: pkg.PkgPath,
				ParseFlag:   ParseFlag(parseFlag),
				FileSet:     fset,
			}
			result.Files[pkg.Syntax[i]] = info
		}

		if s.parseDependency > 0 {
			imports := make([]*packages.Package, 0, len(pkg.Imports))
			for _, dep := range pkg.Imports {
				imports = append(imports, dep)
			}
			if err := s.walkPackagesInternal(imports, fset, result, rootPkgs, pkgSeen); err != nil {
				return err
			}
		}
	}
	return nil
}

func contains(pkgs []*packages.Package, pkg *packages.Package) bool {
	for _, p := range pkgs {
		if p == pkg {
			return true
		}
	}
	return false
}
