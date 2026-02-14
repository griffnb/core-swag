package loader

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// LoadSearchDirs loads Go files from the specified search directories
func (s *Service) LoadSearchDirs(dirs []string) (*LoadResult, error) {
	result := &LoadResult{
		Files: make(map[*ast.File]*AstFileInfo),
	}

	for _, searchDir := range dirs {
		absDir, err := filepath.Abs(searchDir)
		if err != nil {
			return nil, err
		}

		packageDir, err := getPkgName(absDir)
		if err != nil {
			s.debug.Printf("warning: failed to get package name in dir: %s, error: %s", absDir, err.Error())
			packageDir = ""
		}

		err = s.walkDirectory(packageDir, absDir, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// walkDirectory walks a directory and parses Go files
func (s *Service) walkDirectory(packageDir, searchDir string, result *LoadResult) error {
	if s.skipPackageByPrefix(packageDir) {
		return nil
	}

	return filepath.Walk(searchDir, func(path string, f os.FileInfo, wError error) error{
		if wError != nil {
			return fmt.Errorf("failed to access path %q, err: %v", path, wError)
		}

		err := s.shouldSkipDir(path, f)
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		if s.shouldSkipFile(path) {
			return nil
		}

		relPath, err := filepath.Rel(searchDir, path)
		if err != nil {
			return err
		}

		pkgPath := filepath.ToSlash(filepath.Dir(filepath.Clean(filepath.Join(packageDir, relPath))))
		return s.parseFile(pkgPath, path, nil, ParseAll, result)
	})
}

// parseFile parses a single Go file
func (s *Service) parseFile(packageDir, path string, src interface{}, flag ParseFlag, result *LoadResult) error {
	if s.shouldSkipFile(path) {
		return nil
	}

	fileSet := token.NewFileSet()
	astFile, err := parseGoFile(fileSet, path, src)
	if err != nil {
		return fmt.Errorf("failed to parse file %s, error:%+v", path, err)
	}

	info := &AstFileInfo{
		File:        astFile,
		Path:        path,
		PackagePath: packageDir,
		ParseFlag:   flag,
		FileSet:     fileSet,
	}

	result.Files[astFile] = info
	return nil
}

// shouldSkipFile checks if a file should be skipped
func (s *Service) shouldSkipFile(path string) bool {
	if strings.HasSuffix(strings.ToLower(path), "_test.go") {
		return true
	}
	if filepath.Ext(path) != s.parseExtension {
		return true
	}
	return false
}

// shouldSkipDir checks if a directory should be skipped
func (s *Service) shouldSkipDir(path string, f os.FileInfo) error {
	if !f.IsDir() {
		return nil
	}

	if !s.parseVendor && f.Name() == "vendor" {
		return filepath.SkipDir
	}
	if f.Name() == "docs" {
		return filepath.SkipDir
	}
	if len(f.Name()) > 1 && f.Name()[0] == '.' && f.Name() != ".." {
		return filepath.SkipDir
	}

	if s.excludes != nil {
		if _, ok := s.excludes[path]; ok {
			return filepath.SkipDir
		}
	}

	return nil
}

// skipPackageByPrefix checks if a package should be skipped based on prefix
func (s *Service) skipPackageByPrefix(pkgpath string) bool {
	if len(s.packagePrefix) == 0 {
		return false
	}
	for _, prefix := range s.packagePrefix {
		if strings.HasPrefix(pkgpath, prefix) {
			return false
		}
	}
	return true
}
