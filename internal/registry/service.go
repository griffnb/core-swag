// Package registry provides centralized management of type and package registries.
// It handles type discovery, registration, and lookup across Go packages.
package registry

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/griffnb/core-swag/internal/domain"
	"golang.org/x/tools/go/packages"
)

// Service manages package and type registries for swagger documentation generation.
type Service struct {
	files             map[*ast.File]*domain.AstFileInfo
	packages          map[string]*domain.PackageDefinitions
	uniqueDefinitions map[string]*domain.TypeSpecDef
	parseDependency   domain.ParseFlag
	debug             Debugger
}

// NewService creates a new registry service.
func NewService() *Service {
	return &Service{
		files:             make(map[*ast.File]*domain.AstFileInfo),
		packages:          make(map[string]*domain.PackageDefinitions),
		uniqueDefinitions: make(map[string]*domain.TypeSpecDef),
	}
}

// SetParseDependency sets the parse dependency flag.
func (s *Service) SetParseDependency(flag domain.ParseFlag) {
	s.parseDependency = flag
}

// SetDebugger sets the debugger.
func (s *Service) SetDebugger(debug Debugger) {
	s.debug = debug
}

// ParseFile parses a source file.
func (s *Service) ParseFile(packageDir, path string, src interface{}, flag domain.ParseFlag) error {
	fileSet := token.NewFileSet()
	astFile, err := goparser.ParseFile(fileSet, path, src, goparser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s, error:%+v", path, err)
	}
	return s.CollectAstFile(fileSet, packageDir, path, astFile, flag)
}

// CollectAstFile collects an ast.File.
func (s *Service) CollectAstFile(fileSet *token.FileSet, packageDir, path string, astFile *ast.File, flag domain.ParseFlag) error {
	if s.files == nil {
		s.files = make(map[*ast.File]*domain.AstFileInfo)
	}

	if s.packages == nil {
		s.packages = make(map[string]*domain.PackageDefinitions)
	}

	// return without storing the file if we lack a packageDir
	if packageDir == "" {
		return nil
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	dependency, ok := s.packages[packageDir]
	if ok {
		// return without storing the file if it already exists
		_, exists := dependency.Files[path]
		if exists {
			return nil
		}

		dependency.Files[path] = astFile
	} else {
		s.packages[packageDir] = domain.NewPackageDefinitions(astFile.Name.Name, packageDir).AddFile(path, astFile)
	}

	s.files[astFile] = &domain.AstFileInfo{
		FileSet:     fileSet,
		File:        astFile,
		Path:        path,
		PackagePath: packageDir,
		ParseFlag:   flag,
	}

	return nil
}

// RangeFiles iterates over files in alphabetic order.
func (s *Service) RangeFiles(handle func(info *domain.AstFileInfo) error) error {
	sortedFiles := make([]*domain.AstFileInfo, 0, len(s.files))
	for _, info := range s.files {
		// ignore package path prefix with 'vendor' or $GOROOT,
		// because the router info of api will not be included these files.
		if strings.HasPrefix(info.PackagePath, "vendor") ||
			(runtime.GOROOT() != "" && strings.HasPrefix(info.Path, runtime.GOROOT()+string(filepath.Separator))) {
			continue
		}
		sortedFiles = append(sortedFiles, info)
	}

	sort.Slice(sortedFiles, func(i, j int) bool {
		return strings.Compare(sortedFiles[i].Path, sortedFiles[j].Path) < 0
	})

	for _, info := range sortedFiles {
		err := handle(info)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddPackages stores packages.Package to registry.
func (s *Service) AddPackages(pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		pkgDef, ok := s.packages[pkg.PkgPath]
		if !ok {
			continue
		}
		if pkgDef.Package != nil {
			continue
		}
		pkgDef.Package = pkg
		imports := make([]*packages.Package, 0, len(pkg.Imports))
		for _, dep := range pkg.Imports {
			imports = append(imports, dep)
		}
		s.AddPackages(imports)
	}
}

// UniqueDefinitions returns the unique type definitions map.
func (s *Service) UniqueDefinitions() map[string]*domain.TypeSpecDef {
	return s.uniqueDefinitions
}

// Packages returns the packages map.
func (s *Service) Packages() map[string]*domain.PackageDefinitions {
	return s.packages
}

// Files returns the files map.
func (s *Service) Files() map[*ast.File]*domain.AstFileInfo {
	return s.files
}
