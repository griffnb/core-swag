package registry

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/griffnb/core-swag/internal/domain"
)

func TestNewService(t *testing.T) {
	t.Run("creates new service with empty maps", func(t *testing.T) {
		// Act
		svc := NewService()

		// Assert
		if svc == nil {
			t.Fatal("expected service to not be nil")
		}
		if svc.packages == nil {
			t.Error("expected packages map to be initialized")
		}
		if svc.uniqueDefinitions == nil {
			t.Error("expected uniqueDefinitions map to be initialized")
		}
		if svc.files == nil {
			t.Error("expected files map to be initialized")
		}
	})
}

func TestService_CollectAstFile(t *testing.T) {
	t.Run("collects ast file successfully", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
}`
		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}

		// Act
		err = svc.CollectAstFile(fset, "github.com/test/pkg", "/path/to/test.go", astFile, domain.ParseAll)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(svc.files) != 1 {
			t.Errorf("expected 1 file, got %d", len(svc.files))
		}
		if len(svc.packages) != 1 {
			t.Errorf("expected 1 package, got %d", len(svc.packages))
		}
	})

	t.Run("returns without storing if packageDir is empty", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test`
		fset := token.NewFileSet()
		astFile, _ := parser.ParseFile(fset, "test.go", src, parser.ParseComments)

		// Act
		err := svc.CollectAstFile(fset, "", "/path/to/test.go", astFile, domain.ParseAll)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(svc.files) != 0 {
			t.Errorf("expected 0 files, got %d", len(svc.files))
		}
	})

	t.Run("does not duplicate files", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test`
		fset := token.NewFileSet()
		astFile, _ := parser.ParseFile(fset, "test.go", src, parser.ParseComments)

		// Act - collect same file twice
		_ = svc.CollectAstFile(fset, "github.com/test/pkg", "/path/to/test.go", astFile, domain.ParseAll)
		err := svc.CollectAstFile(fset, "github.com/test/pkg", "/path/to/test.go", astFile, domain.ParseAll)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(svc.files) != 1 {
			t.Errorf("expected 1 file after duplicate collection, got %d", len(svc.files))
		}
	})
}

func TestService_ParseFile(t *testing.T) {
	t.Run("parses file successfully", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
}`

		// Act
		err := svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(svc.files) != 1 {
			t.Errorf("expected 1 file, got %d", len(svc.files))
		}
	})

	t.Run("returns error for invalid syntax", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
` // missing closing brace

		// Act
		err := svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)

		// Assert
		if err == nil {
			t.Error("expected error for invalid syntax")
		}
	})
}

func TestService_RangeFiles(t *testing.T) {
	t.Run("iterates files in alphabetic order", func(t *testing.T) {
		// Arrange
		svc := NewService()
		fset := token.NewFileSet()

		// Add files in reverse order
		astFile1, _ := parser.ParseFile(fset, "c.go", "package test", parser.ParseComments)
		astFile2, _ := parser.ParseFile(fset, "a.go", "package test", parser.ParseComments)
		astFile3, _ := parser.ParseFile(fset, "b.go", "package test", parser.ParseComments)

		_ = svc.CollectAstFile(fset, "github.com/test/pkg", "/path/c.go", astFile1, domain.ParseAll)
		_ = svc.CollectAstFile(fset, "github.com/test/pkg", "/path/a.go", astFile2, domain.ParseAll)
		_ = svc.CollectAstFile(fset, "github.com/test/pkg", "/path/b.go", astFile3, domain.ParseAll)

		// Act
		var paths []string
		err := svc.RangeFiles(func(info *domain.AstFileInfo) error {
			paths = append(paths, info.Path)
			return nil
		})

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(paths) != 3 {
			t.Errorf("expected 3 paths, got %d", len(paths))
		}
		if paths[0] != "/path/a.go" || paths[1] != "/path/b.go" || paths[2] != "/path/c.go" {
			t.Errorf("expected alphabetic order, got %v", paths)
		}
	})

	t.Run("skips vendor packages", func(t *testing.T) {
		// Arrange
		svc := NewService()
		fset := token.NewFileSet()

		astFile1, _ := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
		astFile2, _ := parser.ParseFile(fset, "vendor.go", "package vendor", parser.ParseComments)

		_ = svc.CollectAstFile(fset, "github.com/test/pkg", "/path/test.go", astFile1, domain.ParseAll)
		_ = svc.CollectAstFile(fset, "vendor/pkg", "/path/vendor.go", astFile2, domain.ParseAll)

		// Act
		var count int
		err := svc.RangeFiles(func(info *domain.AstFileInfo) error {
			count++
			return nil
		})

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 file (vendor skipped), got %d", count)
		}
	})
}

func TestService_ParseTypes(t *testing.T) {
	t.Run("parses type definitions from files", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
}
type Status int
const (
	Active Status = 1
	Inactive Status = 2
)
`
		_ = svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)

		// Act
		schemas, err := svc.ParseTypes()

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		// Status type is parsed but may not generate a schema since it's a type alias
		// The important thing is that types are registered
		defs := svc.UniqueDefinitions()
		if len(defs) < 1 {
			t.Errorf("expected at least 1 type definition, got %d", len(defs))
		}
		_ = schemas // schemas may be empty for non-primitive structs
	})

	t.Run("handles unique definitions", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
}`
		_ = svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)

		// Act
		_, err := svc.ParseTypes()

		// Assert
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		defs := svc.UniqueDefinitions()
		if len(defs) < 1 {
			t.Error("expected at least one unique definition")
		}
	})
}

func TestService_FindTypeSpec(t *testing.T) {
	t.Run("finds type spec by name", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
}`
		_ = svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)
		_, _ = svc.ParseTypes()

		// Act
		fset := token.NewFileSet()
		astFile, _ := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
		typeDef := svc.FindTypeSpec("User", astFile)

		// Assert
		if typeDef == nil {
			t.Error("expected to find User type")
		}
		if typeDef != nil && typeDef.Name() != "User" {
			t.Errorf("expected User, got %s", typeDef.Name())
		}
	})

	t.Run("returns nil for primitive types", func(t *testing.T) {
		// Arrange
		svc := NewService()

		// Act
		typeDef := svc.FindTypeSpec("string", nil)

		// Assert
		if typeDef != nil {
			t.Error("expected nil for primitive type")
		}
	})

	t.Run("returns nil for non-existent type", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test`
		_ = svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)

		// Act
		fset := token.NewFileSet()
		astFile, _ := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
		typeDef := svc.FindTypeSpec("NonExistent", astFile)

		// Assert
		if typeDef != nil {
			t.Error("expected nil for non-existent type")
		}
	})
}

func TestService_UniqueDefinitions(t *testing.T) {
	t.Run("returns all unique type definitions", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
}
type Product struct {
	Title string
}`
		_ = svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)
		_, _ = svc.ParseTypes()

		// Act
		defs := svc.UniqueDefinitions()

		// Assert
		if len(defs) < 2 {
			t.Errorf("expected at least 2 definitions, got %d", len(defs))
		}
	})
}

func TestService_Packages(t *testing.T) {
	t.Run("returns all registered packages", func(t *testing.T) {
		// Arrange
		svc := NewService()
		src := `package test
type User struct {
	Name string
}`
		_ = svc.ParseFile("github.com/test/pkg", "test.go", src, domain.ParseAll)

		// Act
		pkgs := svc.Packages()

		// Assert
		if len(pkgs) != 1 {
			t.Errorf("expected 1 package, got %d", len(pkgs))
		}
	})
}
