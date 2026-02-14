package loader

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadSearchDirs tests loading Go files from search directories
func TestLoadSearchDirs(t *testing.T) {
	t.Run("loads basic directory", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata/alias_import"
		service := NewService()

		// Act
		result, err := service.LoadSearchDirs([]string{testDir})

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if len(result.Files) == 0 {
			t.Error("expected files to be loaded")
		}
	})

	t.Run("excludes vendor directories", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata"
		service := NewService(WithParseVendor(false))

		// Act
		result, err := service.LoadSearchDirs([]string{testDir})

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// Verify no vendor paths are in result
		for _, info := range result.Files {
			if info != nil && filepath.Base(filepath.Dir(info.Path)) == "vendor" {
				t.Errorf("vendor directory should be excluded: %s", info.Path)
			}
		}
	})

	t.Run("respects package prefix filter", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata/alias_import"
		service := NewService(WithPackagePrefix([]string{"github.com/swaggo/swag/testdata"}))

		// Act
		result, err := service.LoadSearchDirs([]string{testDir})

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
	})

	t.Run("handles non-existent directory", func(t *testing.T) {
		// Arrange
		service := NewService()

		// Act
		_, err := service.LoadSearchDirs([]string{"/non/existent/path"})

		// Assert
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})

	t.Run("handles multiple search directories", func(t *testing.T) {
		// Arrange
		testDir1 := "../../testdata/alias_import"
		testDir2 := "../../testdata/alias_type"
		service := NewService()

		// Act
		result, err := service.LoadSearchDirs([]string{testDir1, testDir2})

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result.Files) == 0 {
			t.Error("expected files from multiple directories")
		}
	})
}

// TestLoadDependencies tests dependency loading with depth parameter
func TestLoadDependencies(t *testing.T) {
	t.Run("loads with go list", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata/alias_import"
		service := NewService(WithGoList(true), WithParseDependency(ParseModels))

		// Act
		result, err := service.LoadDependencies([]string{testDir}, 10)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
	})

	t.Run("loads with depth package", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata/alias_import"
		service := NewService(WithGoList(false), WithParseDependency(ParseModels))

		// Act
		result, err := service.LoadDependencies([]string{testDir}, 1)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
	})

	t.Run("skips internal packages when configured", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata"
		service := NewService(
			WithParseInternal(false),
			WithParseDependency(ParseModels),
		)

		// Act
		result, err := service.LoadDependencies([]string{testDir}, 1)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
	})

	t.Run("respects parse depth limit", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata/alias_import"
		service := NewService(WithParseDependency(ParseModels))

		// Act
		result1, err := service.LoadDependencies([]string{testDir}, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		result2, err := service.LoadDependencies([]string{testDir}, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Assert - deeper depth should potentially load more files
		if result1 == nil || result2 == nil {
			t.Fatal("expected results, got nil")
		}
	})
}

// TestGoPackagesIntegration tests go/packages integration
func TestGoPackagesIntegration(t *testing.T) {
	t.Run("loads packages with go/packages", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata/code_examples"
		mainFile := filepath.Join(testDir, "main.go")
		service := NewService(WithGoPackages(true))

		// Act
		result, err := service.LoadWithGoPackages([]string{testDir}, mainFile)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if len(result.Packages) == 0 {
			t.Error("expected packages to be loaded")
		}
	})

	t.Run("loads dependencies with go/packages", func(t *testing.T) {
		// Arrange
		testDir := "../../testdata/code_examples"
		mainFile := filepath.Join(testDir, "main.go")
		service := NewService(
			WithGoPackages(true),
			WithParseDependency(ParseModels),
		)

		// Act
		result, err := service.LoadWithGoPackages([]string{testDir}, mainFile)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
	})

	t.Run("handles package errors", func(t *testing.T) {
		// Arrange
		service := NewService(WithGoPackages(true))

		// Act
		_, err := service.LoadWithGoPackages([]string{"/invalid/path"}, "/invalid/main.go")

		// Assert
		if err == nil {
			t.Error("expected error for invalid path")
		}
	})
}

// TestExcludePatterns tests vendor/internal exclusion logic
func TestExcludePatterns(t *testing.T) {
	t.Run("excludes vendor when configured", func(t *testing.T) {
		// Arrange
		service := NewService(WithParseVendor(false))

		// Create temp test structure
		tmpDir := t.TempDir()
		vendorDir := filepath.Join(tmpDir, "vendor")
		err := os.Mkdir(vendorDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Act
		result, err := service.LoadSearchDirs([]string{tmpDir})

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, info := range result.Files {
			if info != nil && filepath.Base(filepath.Dir(info.Path)) == "vendor" {
				t.Error("vendor should be excluded")
			}
		}
	})

	t.Run("includes vendor when configured", func(t *testing.T) {
		// Arrange
		service := NewService(WithParseVendor(true))

		// This test validates the option is set correctly
		if !service.parseVendor {
			t.Error("parseVendor should be true")
		}
	})

	t.Run("respects custom exclude patterns", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		excludeDir := filepath.Join(tmpDir, "excluded")
		err := os.Mkdir(excludeDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		service := NewService(WithExcludes(map[string]struct{}{
			excludeDir: {},
		}))

		// Act
		result, err := service.LoadSearchDirs([]string{tmpDir})

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, info := range result.Files {
			if info != nil && filepath.Dir(info.Path) == excludeDir {
				t.Error("excluded directory should be skipped")
			}
		}
	})
}

// TestOptions tests service configuration options
func TestOptions(t *testing.T) {
	t.Run("applies all options correctly", func(t *testing.T) {
		// Arrange & Act
		service := NewService(
			WithParseVendor(true),
			WithParseInternal(true),
			WithGoList(true),
			WithGoPackages(true),
			WithPackagePrefix([]string{"github.com/test"}),
			WithParseExtension(".go"),
			WithParseDependency(ParseAll),
			WithExcludes(map[string]struct{}{"test": {}}),
			WithDebugger(&testDebugger{}),
		)

		// Assert
		if !service.parseVendor {
			t.Error("parseVendor should be true")
		}
		if !service.parseInternal {
			t.Error("parseInternal should be true")
		}
		if !service.useGoList {
			t.Error("useGoList should be true")
		}
		if !service.useGoPackages {
			t.Error("useGoPackages should be true")
		}
		if len(service.packagePrefix) != 1 {
			t.Error("packagePrefix should have one entry")
		}
		if service.parseExtension != ".go" {
			t.Error("parseExtension should be .go")
		}
		if service.parseDependency != ParseAll {
			t.Error("parseDependency should be ParseAll")
		}
		if len(service.excludes) != 1 {
			t.Error("excludes should have one entry")
		}
		if service.debug == nil {
			t.Error("debug should be set")
		}
	})
}

// testDebugger is a mock debugger for testing
type testDebugger struct{}

func (t *testDebugger) Printf(format string, v ...interface{}) {}

// TestSkipLogic tests file and directory skip logic
func TestSkipLogic(t *testing.T) {
	t.Run("skips test files", func(t *testing.T) {
		// Arrange
		service := NewService()

		// Act
		shouldSkip := service.shouldSkipFile("some_test.go")

		// Assert
		if !shouldSkip {
			t.Error("should skip test files")
		}
	})

	t.Run("skips non-go files", func(t *testing.T) {
		// Arrange
		service := NewService()

		// Act
		shouldSkip := service.shouldSkipFile("readme.md")

		// Assert
		if !shouldSkip {
			t.Error("should skip non-go files")
		}
	})

	t.Run("processes regular go files", func(t *testing.T) {
		// Arrange
		service := NewService()

		// Act
		shouldSkip := service.shouldSkipFile("main.go")

		// Assert
		if shouldSkip {
			t.Error("should not skip regular go files")
		}
	})

	t.Run("skips hidden directories", func(t *testing.T) {
		// Arrange
		service := NewService()
		fileInfo := &mockFileInfo{name: ".git", isDir: true}

		// Act
		err := service.shouldSkipDir("/some/path/.git", fileInfo)

		// Assert
		if err == nil {
			t.Error("should skip hidden directories")
		}
	})

	t.Run("skips docs directory", func(t *testing.T) {
		// Arrange
		service := NewService()
		fileInfo := &mockFileInfo{name: "docs", isDir: true}

		// Act
		err := service.shouldSkipDir("/some/path/docs", fileInfo)

		// Assert
		if err == nil {
			t.Error("should skip docs directory")
		}
	})
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// TestAstFileInfo tests the AstFileInfo structure
func TestAstFileInfo(t *testing.T) {
	t.Run("creates valid AstFileInfo", func(t *testing.T) {
		// Arrange
		info := &AstFileInfo{
			File:        &ast.File{},
			Path:        "/test/path.go",
			PackagePath: "github.com/test/pkg",
			ParseFlag:   ParseAll,
		}

		// Assert
		if info.File == nil {
			t.Error("File should not be nil")
		}
		if info.Path != "/test/path.go" {
			t.Error("Path should match")
		}
		if info.PackagePath != "github.com/test/pkg" {
			t.Error("PackagePath should match")
		}
		if info.ParseFlag != ParseAll {
			t.Error("ParseFlag should match")
		}
	})
}
