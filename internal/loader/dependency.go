package loader

import (
	"context"
	"fmt"
	"go/ast"
	"go/build"
	"os"
	"path/filepath"

	"github.com/KyleBanks/depth"
)

// LoadDependencies loads package dependencies up to the specified depth
func (s *Service) LoadDependencies(dirs []string, maxDepth int) (*LoadResult, error) {
	if s.parseDependency == ParseNone {
		return &LoadResult{Files: make(map[*ast.File]*AstFileInfo)}, nil
	}

	result := &LoadResult{
		Files: make(map[*ast.File]*AstFileInfo),
	}

	if s.useGoList {
		return s.loadDependenciesWithGoList(dirs, result)
	}

	return s.loadDependenciesWithDepth(dirs, maxDepth, result)
}

// loadDependenciesWithGoList uses go list to load dependencies
func (s *Service) loadDependenciesWithGoList(dirs []string, result *LoadResult) (*LoadResult, error) {
	pkgs, err := listPackages(context.Background(), dirs, nil, "-deps")
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		err := s.loadPackageFromList(pkg, s.parseDependency, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// loadDependenciesWithDepth uses depth package to load dependencies
func (s *Service) loadDependenciesWithDepth(dirs []string, maxDepth int, result *LoadResult) (*LoadResult, error) {
	dirImported := make(map[string]struct{})
	for _, dir := range dirs {
		absDir, err := filepath.Abs(dir)
		if err == nil {
			dirImported[absDir] = struct{}{}
		}
	}

	for index, dir := range dirs {
		var t depth.Tree
		t.ResolveInternal = true
		t.MaxDepth = maxDepth

		pkgName, err := getPkgName(dir)
		if err != nil {
			if index == 0 {
				return nil, err
			}
			continue
		}

		err = t.Resolve(pkgName)
		if err != nil {
			return nil, fmt.Errorf("pkg %s cannot find all dependencies, %s", pkgName, err)
		}

		for i := 0; i < len(t.Root.Deps); i++ {
			err := s.loadFromDepth(&t.Root.Deps[i], s.parseDependency, dirImported, result)
			if err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// loadPackageFromList loads a package from go list
func (s *Service) loadPackageFromList(pkg *build.Package, parseFlag ParseFlag, result *LoadResult) error {
	ignoreInternal := pkg.Goroot && !s.parseInternal
	if ignoreInternal {
		return nil
	}

	if s.skipPackageByPrefix(pkg.ImportPath) {
		return nil
	}

	srcDir := pkg.Dir
	for i := range pkg.GoFiles {
		err := s.parseFile(pkg.ImportPath, filepath.Join(srcDir, pkg.GoFiles[i]), nil, parseFlag, result)
		if err != nil {
			return err
		}
	}

	for i := range pkg.CgoFiles {
		err := s.parseFile(pkg.ImportPath, filepath.Join(srcDir, pkg.CgoFiles[i]), nil, parseFlag, result)
		if err != nil {
			return err
		}
	}

	return nil
}

// loadFromDepth loads dependencies from depth tree
func (s *Service) loadFromDepth(pkg *depth.Pkg, parseFlag ParseFlag, dirImported map[string]struct{}, result *LoadResult) error {
	ignoreInternal := pkg.Internal && !s.parseInternal
	if ignoreInternal || !pkg.Resolved {
		return nil
	}

	if pkg.Raw != nil && s.skipPackageByPrefix(pkg.Raw.ImportPath) {
		return nil
	}

	if pkg.Raw == nil && pkg.Name == "C" {
		return nil
	}

	srcDir := pkg.Raw.Dir
	if _, ok := dirImported[srcDir]; ok {
		return nil
	}
	dirImported[srcDir] = struct{}{}

	files, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		path := filepath.Join(srcDir, f.Name())
		if err := s.parseFile(pkg.Name, path, nil, parseFlag, result); err != nil {
			return err
		}
	}

	for i := 0; i < len(pkg.Deps); i++ {
		if err := s.loadFromDepth(&pkg.Deps[i], parseFlag, dirImported, result); err != nil {
			return err
		}
	}

	return nil
}
