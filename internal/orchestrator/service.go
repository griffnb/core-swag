// Package orchestrator coordinates all services to generate OpenAPI documentation.
// It provides a clean, simple coordinator that delegates to specialized services.
package orchestrator

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/loader"
	"github.com/griffnb/core-swag/internal/model"
	"github.com/griffnb/core-swag/internal/parser/base"
	"github.com/griffnb/core-swag/internal/parser/route"
	"github.com/griffnb/core-swag/internal/registry"
	"github.com/griffnb/core-swag/internal/schema"
)

// Service coordinates all parsing services to generate OpenAPI documentation.
type Service struct {
	loader        *loader.Service
	registry      *registry.Service
	schemaBuilder *schema.BuilderService
	baseParser    *base.Service
	routeParser   *route.Service
	swagger       *spec.Swagger
	config        *Config
}

// Config holds orchestrator configuration options.
type Config struct {
	ParseVendor             bool
	ParseInternal           bool
	ParseDependency         loader.ParseFlag
	PropNamingStrategy      string
	RequiredByDefault       bool
	Strict                  bool
	MarkdownFileDir         string
	CodeExampleFilesDir     string
	CollectionFormatInQuery string
	Excludes                map[string]struct{}
	PackagePrefix           []string
	ParseExtension          string
	ParseGoList             bool
	ParseGoPackages         bool
	HostState               string
	ParseFuncBody           bool
	UseStructName           bool
	Overrides               map[string]string
	Tags                    map[string]struct{}
	Debug                   Debugger
}

// Debugger is the interface for debug logging.
type Debugger interface {
	Printf(format string, v ...interface{})
}

// New creates a new orchestrator service with the given configuration.
func New(config *Config) *Service {
	if config == nil {
		config = &Config{}
	}

	// Apply defaults for zero values
	if config.PropNamingStrategy == "" {
		config.PropNamingStrategy = "camelcase"
	}
	if config.CollectionFormatInQuery == "" {
		config.CollectionFormatInQuery = "csv"
	}
	if config.Excludes == nil {
		config.Excludes = make(map[string]struct{})
	}
	if config.PackagePrefix == nil {
		config.PackagePrefix = []string{}
	}
	if config.ParseExtension == "" {
		config.ParseExtension = ".go"
	}
	if config.Overrides == nil {
		config.Overrides = make(map[string]string)
	}
	if config.Tags == nil {
		config.Tags = make(map[string]struct{})
	}
	// Note: ParseInternal defaults to false (zero value)
	// Note: ParseDependency defaults to ParseNone (zero value)
	// Note: ParseFuncBody defaults to false (zero value)

	// Create loader service
	loaderService := loader.NewService(
		loader.WithParseVendor(config.ParseVendor),
		loader.WithParseInternal(config.ParseInternal),
		loader.WithParseDependency(config.ParseDependency),
		loader.WithExcludes(config.Excludes),
		loader.WithPackagePrefix(config.PackagePrefix),
		loader.WithParseExtension(config.ParseExtension),
		loader.WithGoList(config.ParseGoList),
		loader.WithGoPackages(config.ParseGoPackages),
		loader.WithDebugger(config.Debug),
	)

	// Create registry service
	registryService := registry.NewService()
	registryService.SetParseDependency(config.ParseDependency)
	if config.Debug != nil {
		registryService.SetDebugger(config.Debug)
	}

	// Create schema builder
	schemaBuilder := schema.NewBuilder()
	schemaBuilder.SetPropNamingStrategy(config.PropNamingStrategy)
	schemaBuilder.SetTypeResolver(registryService) // Enable type alias resolution

	// Create and configure CoreStructParser for proper field resolution
	coreStructParser := &model.CoreStructParser{}
	schemaBuilder.SetStructParser(coreStructParser)

	// Create enum lookup using CoreStructParser
	enumLookup := &model.ParserEnumLookup{
		Parser: coreStructParser,
	}
	schemaBuilder.SetEnumLookup(enumLookup)

	// Create swagger spec
	swagger := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Contact: &spec.ContactInfo{},
					License: nil,
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: spec.Extensions{},
				},
			},
			Paths:               &spec.Paths{Paths: make(map[string]spec.PathItem)},
			Definitions:         make(spec.Definitions),
			SecurityDefinitions: make(spec.SecurityDefinitions),
		},
	}

	// Create base parser
	baseParser := base.NewService(swagger)
	if config.MarkdownFileDir != "" {
		baseParser.SetMarkdownFileDir(config.MarkdownFileDir)
	}
	if config.Debug != nil {
		baseParser.SetDebugger(config.Debug)
	}

	// Create route parser
	// Note: Passing nil for type resolver - routes will use basic type schemas
	// TODO: Implement proper type resolution adapter when needed
	routeParser := route.NewService(nil, config.CollectionFormatInQuery)
	if config.MarkdownFileDir != "" {
		routeParser.SetMarkdownFileDir(config.MarkdownFileDir)
	}
	// Inject registry for @NoPublic annotation support
	routeParser.SetRegistry(registryService)

	return &Service{
		loader:        loaderService,
		registry:      registryService,
		schemaBuilder: schemaBuilder,
		baseParser:    baseParser,
		routeParser:   routeParser,
		swagger:       swagger,
		config:        config,
	}
}

// Parse generates OpenAPI documentation from the given search directories and main API file.
// This is the main entry point that coordinates all services.
func (s *Service) Parse(searchDirs []string, mainAPIFile string, parseDepth int) (*spec.Swagger, error) {
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Starting parse with %d search dirs", len(searchDirs))
	}

	// Step 1: Load packages and files
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Step 1 - Loading packages")
	}

	var loadResult *loader.LoadResult
	var err error

	if s.config.ParseGoPackages {
		// Use go/packages API (most robust)
		loadResult, err = s.loader.LoadWithGoPackages(searchDirs, mainAPIFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load packages with go/packages: %w", err)
		}
	} else {
		// Use directory walking
		loadResult, err = s.loader.LoadSearchDirs(searchDirs)
		if err != nil {
			return nil, fmt.Errorf("failed to load search directories: %w", err)
		}

		// Load dependencies if needed
		if parseDepth > 0 && s.config.ParseDependency != loader.ParseNone {
			depResult, err := s.loader.LoadDependencies(searchDirs, parseDepth)
			if err != nil {
				return nil, fmt.Errorf("failed to load dependencies: %w", err)
			}
			// Merge results
			for astFile, info := range depResult.Files {
				loadResult.Files[astFile] = info
			}
		}
	}

	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Loaded %d files", len(loadResult.Files))
	}

	// Step 1b: Seed downstream caches from loaded packages to eliminate redundant packages.Load() calls
	if loadResult.Packages != nil {
		if s.config.Debug != nil {
			s.config.Debug.Printf("Orchestrator: Step 1b - Seeding package caches from %d top-level packages", len(loadResult.Packages))
		}
		model.SeedGlobalPackageCache(loadResult.Packages)
		model.SeedEnumPackageCache(loadResult.Packages)
	}

	// Step 2: Register types with registry
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Step 2 - Registering types")
	}

	// Collect files into registry
	for astFile, fileInfo := range loadResult.Files {
		err = s.registry.CollectAstFile(
			fileInfo.FileSet,
			fileInfo.PackagePath,
			fileInfo.Path,
			astFile,
			fileInfo.ParseFlag,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to collect AST file %s: %w", fileInfo.Path, err)
		}
	}

	// Parse types in registry
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Parsing types in registry")
	}
	schemas, err := s.registry.ParseTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to parse types: %w", err)
	}
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Parsed %d schemas from registry", len(schemas))
	}

	// Set global name resolver so CoreStructParser produces short $ref names for
	// unique types and full-path names for NotUnique types. Must happen after
	// ParseTypes() which sets the NotUnique flags.
	model.SetGlobalNameResolver(newRegistryNameResolver(s.registry))

	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Registry has %d unique definitions", len(s.registry.UniqueDefinitions()))
	}

	// Step 3: Parse general API info from main file
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Step 3 - Parsing general API info")
	}

	// The mainAPIFile parameter can be:
	// 1. Relative to searchDir (e.g., "main.go" or "./main.go")
	// 2. Absolute path
	// 3. Relative to CWD
	// We need to find it relative to searchDir if it's just a filename
	mainFilePath := mainAPIFile
	if !filepath.IsAbs(mainAPIFile) {
		// Check if it's just a filename (no directory component)
		if filepath.Base(mainAPIFile) == mainAPIFile || filepath.Dir(mainAPIFile) == "." {
			// It's relative to searchDir - join with searchDir
			if len(searchDirs) > 0 {
				mainFilePath = filepath.Join(searchDirs[0], mainAPIFile)
			}
		}
		// Otherwise, it's already a path relative to CWD, use as-is
	}

	err = s.baseParser.ParseGeneralAPIInfo(mainFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse general API info: %w", err)
	}

	// Step 4: Parse routes from all files (parallel)
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Step 4 - Parsing routes (parallel, limit=%d)", runtime.NumCPU())
	}

	allRoutes, routeCount, err := s.parseRoutesParallel(loadResult.Files)
	if err != nil {
		return nil, err
	}

	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Parsed %d routes", routeCount)
	}

	// Step 5: Build schemas (demand-driven)
	// Only build schemas for types referenced by routes, not all 60K+ registry types.
	// BuildAllSchemas handles Public variants and transitive nested dependencies.
	referencedTypes := CollectReferencedTypes(allRoutes)
	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Step 5 - Building schemas (demand-driven, %d route-referenced types)",
			len(referencedTypes))
	}

	err = s.buildDemandDrivenSchemas(referencedTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to build demand-driven schemas: %w", err)
	}

	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Built %d schema definitions", len(s.swagger.Definitions))
	}

	if s.config.Debug != nil {
		hits, misses := model.GlobalCacheStats()
		s.config.Debug.Printf("Orchestrator: Package cache hits=%d misses=%d", hits, misses)
	}

	// Step 6: Cleanup unused definitions
	// TODO: Implement cleanup logic
	// For now, keep all definitions

	if s.config.Debug != nil {
		s.config.Debug.Printf("Orchestrator: Parse complete")
	}

	return s.swagger, nil
}

// GetSwagger returns the swagger specification.
// This provides access to the swagger spec for backward compatibility.
func (s *Service) GetSwagger() *spec.Swagger {
	return s.swagger
}

// Registry returns the registry service for external access.
func (s *Service) Registry() *registry.Service {
	return s.registry
}

// SchemaBuilder returns the schema builder service for external access.
func (s *Service) SchemaBuilder() *schema.BuilderService {
	return s.schemaBuilder
}
