package orchestrator

import (
	"log"
	"os"
	"testing"

	"github.com/griffnb/core-swag/internal/loader"
)

func TestOrchestratorBasic(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	config := &Config{
		ParseVendor:             false,
		ParseInternal:           true,
		ParseDependency:         loader.ParseModels,
		PropNamingStrategy:      "camelcase",
		RequiredByDefault:       false,
		Strict:                  false,
		MarkdownFileDir:         "",
		CodeExampleFilesDir:     "",
		CollectionFormatInQuery: "csv",
		Excludes:                make(map[string]struct{}),
		PackagePrefix:           []string{},
		ParseExtension:          ".go",
		ParseGoList:             true,  // Try with go list
		ParseGoPackages:         false,
		HostState:               "",
		ParseFuncBody:           true,
		UseStructName:           false,
		Overrides:               make(map[string]string),
		Tags:                    make(map[string]struct{}),
		Debug:                   logger,
	}

	orc := New(config)

	// Parse testdata/delims
	searchDirs := []string{"../../testdata/delims"}
	mainFile := "main.go"
	parseDepth := 100

	swagger, err := orc.Parse(searchDirs, mainFile, parseDepth)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Swagger Info: %+v", swagger.Info)
	t.Logf("Paths count: %d", len(swagger.Paths.Paths))

	for path, pathItem := range swagger.Paths.Paths {
		t.Logf("Path: %s", path)
		if pathItem.Get != nil {
			t.Logf("  GET: %s", pathItem.Get.Description)
		}
	}

	t.Logf("Definitions count: %d", len(swagger.Definitions))
	for name, def := range swagger.Definitions {
		t.Logf("Definition %s: type=%v, properties=%d", name, def.Type, len(def.Properties))
	}

	if len(swagger.Paths.Paths) == 0 {
		t.Error("No paths parsed!")
	}

	// Check registry
	typeDefs := orc.Registry().UniqueDefinitions()
	t.Logf("Registry has %d type definitions", len(typeDefs))
	for _, typeDef := range typeDefs {
		if typeDef.TypeName() == "api.MyStruct" || typeDef.TypeName() == "MyStruct" {
			t.Logf("Found MyStruct in registry: %s", typeDef.TypeName())
		}
	}
}
