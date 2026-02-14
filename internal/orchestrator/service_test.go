package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/swaggo/swag/internal/loader"
)

func TestNew(t *testing.T) {
	t.Run("creates orchestrator with default config", func(t *testing.T) {
		// Act
		service := New(nil)

		// Assert
		if service == nil {
			t.Fatal("expected service to be non-nil")
		}
		if service.loader == nil {
			t.Error("expected loader to be initialized")
		}
		if service.registry == nil {
			t.Error("expected registry to be initialized")
		}
		if service.schemaBuilder == nil {
			t.Error("expected schema builder to be initialized")
		}
		if service.baseParser == nil {
			t.Error("expected base parser to be initialized")
		}
		if service.swagger == nil {
			t.Error("expected swagger to be initialized")
		}
		if service.config == nil {
			t.Error("expected config to be initialized")
		}
	})

	t.Run("creates orchestrator with custom config", func(t *testing.T) {
		// Arrange
		config := &Config{
			ParseVendor:        true,
			ParseInternal:      false,
			ParseDependency:    loader.ParseAll,
			PropNamingStrategy: "snakecase",
		}

		// Act
		service := New(config)

		// Assert
		if service == nil {
			t.Fatal("expected service to be non-nil")
		}
		if service.config != config {
			t.Error("expected config to match")
		}
		if service.config.ParseVendor != true {
			t.Error("expected ParseVendor to be true")
		}
		if service.config.PropNamingStrategy != "snakecase" {
			t.Error("expected PropNamingStrategy to be snakecase")
		}
	})
}

func TestService_Parse(t *testing.T) {
	t.Run("parses simple API", func(t *testing.T) {
		// Arrange
		testDir := t.TempDir()
		mainFile := filepath.Join(testDir, "main.go")

		mainContent := `package main

// @title Test API
// @version 1.0
// @description This is a test API
// @host localhost:8080
// @basePath /api

func main() {}
`

		err := os.WriteFile(mainFile, []byte(mainContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config := &Config{
			ParseGoPackages: false, // Use simple directory walking for tests
			ParseGoList:     false,
		}
		service := New(config)

		// Act
		swagger, err := service.Parse([]string{testDir}, mainFile, 0)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if swagger == nil {
			t.Fatal("expected swagger to be non-nil")
		}
		if swagger.Info == nil {
			t.Fatal("expected swagger info to be non-nil")
		}
		if swagger.Info.Title != "Test API" {
			t.Errorf("expected title 'Test API', got: %s", swagger.Info.Title)
		}
		if swagger.Info.Version != "1.0" {
			t.Errorf("expected version '1.0', got: %s", swagger.Info.Version)
		}
		if swagger.Host != "localhost:8080" {
			t.Errorf("expected host 'localhost:8080', got: %s", swagger.Host)
		}
		if swagger.BasePath != "/api" {
			t.Errorf("expected basePath '/api', got: %s", swagger.BasePath)
		}
	})

	t.Run("handles empty search dirs", func(t *testing.T) {
		// Arrange
		testDir := t.TempDir()
		mainFile := filepath.Join(testDir, "main.go")

		mainContent := `package main

// @title Test API
// @version 1.0

func main() {}
`

		err := os.WriteFile(mainFile, []byte(mainContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		config := &Config{
			ParseGoPackages: false, // Use simple directory walking for tests
			ParseGoList:     false,
		}
		service := New(config)

		// Act
		swagger, err := service.Parse([]string{testDir}, mainFile, 0)

		// Assert - should not error even with empty directory
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if swagger == nil {
			t.Fatal("expected swagger to be non-nil")
		}
	})
}

func TestService_GetSwagger(t *testing.T) {
	t.Run("returns swagger spec", func(t *testing.T) {
		// Arrange
		service := New(nil)

		// Act
		swagger := service.GetSwagger()

		// Assert
		if swagger == nil {
			t.Fatal("expected swagger to be non-nil")
		}
		if swagger.Info == nil {
			t.Error("expected swagger info to be initialized")
		}
		if swagger.Paths == nil {
			t.Error("expected swagger paths to be initialized")
		}
		if swagger.Definitions == nil {
			t.Error("expected swagger definitions to be initialized")
		}
	})
}

func TestService_Registry(t *testing.T) {
	t.Run("returns registry service", func(t *testing.T) {
		// Arrange
		service := New(nil)

		// Act
		registry := service.Registry()

		// Assert
		if registry == nil {
			t.Fatal("expected registry to be non-nil")
		}
	})
}

func TestService_SchemaBuilder(t *testing.T) {
	t.Run("returns schema builder", func(t *testing.T) {
		// Arrange
		service := New(nil)

		// Act
		builder := service.SchemaBuilder()

		// Assert
		if builder == nil {
			t.Fatal("expected schema builder to be non-nil")
		}
	})
}
