package model

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/console"
	"golang.org/x/tools/go/packages"
)

// Global package cache shared across all parsers
var (
	globalPackageCache = make(map[string]*packages.Package)
	globalCacheMutex   sync.RWMutex
)

// cacheHits and cacheMisses track how often LookupStructFields resolves a
// package from the global cache vs falling back to packages.Load.
var (
	cacheHits   int64
	cacheMisses int64
)

// GlobalCacheStats returns the global package cache hit and miss counts.
func GlobalCacheStats() (hits, misses int64) {
	return atomic.LoadInt64(&cacheHits), atomic.LoadInt64(&cacheMisses)
}

// ResetGlobalCacheStats resets the cache statistics counters for test isolation.
func ResetGlobalCacheStats() {
	atomic.StoreInt64(&cacheHits, 0)
	atomic.StoreInt64(&cacheMisses, 0)
}

// SeedGlobalPackageCache pre-populates the global package cache with all
// packages and their transitive imports. This avoids redundant packages.Load
// calls during struct parsing by warming the cache upfront.
func SeedGlobalPackageCache(pkgs []*packages.Package) {
	if len(pkgs) == 0 {
		return
	}

	visited := make(map[string]bool)

	globalCacheMutex.Lock()
	defer globalCacheMutex.Unlock()

	// Pass 1: Cache all directly loaded packages first — these have full
	// syntax from the initial packages.Load call. This ensures they take
	// priority over syntax-less versions found through transitive imports.
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		globalPackageCache[pkg.PkgPath] = pkg
		visited[pkg.PkgPath] = true
	}

	// Pass 2: Walk transitive imports, caching only packages not already cached
	// with a better (syntax-bearing) version.
	var walk func(pkg *packages.Package)
	walk = func(pkg *packages.Package) {
		if pkg == nil {
			return
		}
		if visited[pkg.PkgPath] {
			return
		}
		visited[pkg.PkgPath] = true

		if globalPackageCache[pkg.PkgPath] == nil {
			globalPackageCache[pkg.PkgPath] = pkg
		}

		for _, imp := range pkg.Imports {
			walk(imp)
		}
	}

	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, imp := range pkg.Imports {
			walk(imp)
		}
	}
}

type CoreStructParser struct {
	basePackage   *packages.Package
	packageMap    map[string]*packages.Package
	visited       map[string]bool
	packageCache  map[string]*packages.Package // Cache loaded packages
	typeCache     map[string]*StructBuilder    // Cache processed types
	cacheMutex    sync.RWMutex                 // Protect caches
	sharedFileSet *token.FileSet               // Shared FileSet for fallback packages.Load calls
}

// getOrCreateFileSet returns the shared FileSet, creating one if needed.
// token.FileSet is internally thread-safe so no additional mutex is required.
func (c *CoreStructParser) getOrCreateFileSet() *token.FileSet {
	if c.sharedFileSet == nil {
		c.sharedFileSet = token.NewFileSet()
	}
	return c.sharedFileSet
}

// toPascalCase converts package_name or package-name to PascalCase (PackageName)
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Split by underscore or hyphen
	var parts []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	}) {
		if len(part) > 0 {
			// Capitalize first letter of each part
			parts = append(parts, strings.ToUpper(part[:1])+part[1:])
		}
	}

	if len(parts) == 0 {
		// No delimiters, just capitalize first letter
		return strings.ToUpper(s[:1]) + s[1:]
	}

	return strings.Join(parts, "")
}

func (c *CoreStructParser) LookupStructFields(_, importPath, typeName string) *StructBuilder {
	// Check type cache first
	cacheKey := importPath + ":" + typeName
	c.cacheMutex.RLock()
	if cached, exists := c.typeCache[cacheKey]; exists {
		c.cacheMutex.RUnlock()
		console.Logger.Debug("Using cached type: %s\n", cacheKey)
		return cached
	}
	c.cacheMutex.RUnlock()

	builder := &StructBuilder{}

	// Initialize caches if needed
	c.cacheMutex.Lock()
	if c.packageCache == nil {
		c.packageCache = make(map[string]*packages.Package)
	}
	if c.typeCache == nil {
		c.typeCache = make(map[string]*StructBuilder)
	}
	c.cacheMutex.Unlock()

	// Check global cache first — but only use it if the package has AST syntax.
	// Packages cached from transitive deps (via SeedGlobalPackageCache) may lack
	// Syntax even though they have TypesInfo, making field extraction impossible.
	globalCacheMutex.RLock()
	pkg, pkgCached := globalPackageCache[importPath]
	globalCacheMutex.RUnlock()

	if pkgCached && len(pkg.Syntax) == 0 {
		console.Logger.Debug("Cached package %s has no syntax, reloading\n", importPath)
		pkgCached = false
	}

	var packageMap map[string]*packages.Package

	if !pkgCached {
		atomic.AddInt64(&cacheMisses, 1)
		console.Logger.Debug("Loading package: %s\n", importPath)
		cfg := &packages.Config{
			Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedName | packages.NeedImports | packages.NeedDeps,
			Fset: c.getOrCreateFileSet(),
		}
		// Load the main package with all its dependencies
		pkgs, err := packages.Load(cfg, importPath)
		if err != nil || len(pkgs) == 0 {
			log.Fatalf("failed to load package %s: %v", importPath, err)
		}
		packageMap = make(map[string]*packages.Package)

		// Recursively add all packages including imports and dependencies
		var addPackage func(*packages.Package)
		addPackage = func(p *packages.Package) {
			if p == nil || packageMap[p.PkgPath] != nil {
				return
			}
			packageMap[p.PkgPath] = p

			// Add all imports
			for _, imp := range p.Imports {
				addPackage(imp)
			}
		}

		for _, p := range pkgs {
			addPackage(p)
		}

		// Cache all loaded packages in both local and global cache
		globalCacheMutex.Lock()
		c.cacheMutex.Lock()
		for path, p := range packageMap {
			globalPackageCache[path] = p
			c.packageCache[path] = p
		}
		pkg = packageMap[importPath]
		c.cacheMutex.Unlock()
		globalCacheMutex.Unlock()
		console.Logger.Debug("Cached %d packages from %s\n", len(packageMap), importPath)
	} else {
		atomic.AddInt64(&cacheHits, 1)
		console.Logger.Debug("Using globally cached package: %s\n", importPath)
		// Use cached packages from global cache
		globalCacheMutex.RLock()
		packageMap = make(map[string]*packages.Package)
		for k, v := range globalPackageCache {
			packageMap[k] = v
		}
		globalCacheMutex.RUnlock()
	}

	// Set the packageMap on the parser so checkNamed can use it
	c.packageMap = packageMap

	if pkg == nil || pkg.PkgPath != importPath {
		console.Logger.Debug("Package not found or mismatch: %v\n", importPath)
		return builder
	}

	console.Logger.Debug("Processing package: %+v %s\n", pkg, typeName)

	// Set basePackage so processStructField can qualify same-package types
	c.basePackage = pkg

	visited := make(map[string]bool)
	c.visited = visited
	fields := c.ExtractFieldsRecursive(pkg, typeName, packageMap, visited)

	// Process all fields
	for _, f := range fields {
		console.Logger.Debug("Field: %s, Type: %s, Tag: %s\n", f.Name, f.Type, f.Tag)

		// Check if it's a special StructField type that needs expansion
		if f.IsGeneric() && strings.Contains(f.EffectiveTypeString(), "fields.StructField") {
			c.processStructField(f, packageMap, builder)
		} else {
			builder.Fields = append(builder.Fields, f)
		}
	}

	// Cache the result before returning
	c.cacheMutex.Lock()
	c.typeCache[cacheKey] = builder
	c.cacheMutex.Unlock()

	return builder
}

// processStructField handles the expansion of StructField[T] types
func (c *CoreStructParser) processStructField(f *StructField, packageMap map[string]*packages.Package, builder *StructBuilder) {
	if !f.IsGeneric() {
		builder.Fields = append(builder.Fields, f)
		return
	}
	subTypeName, err := f.GenericTypeArg()
	if err != nil {
		builder.Fields = append(builder.Fields, f)
		return
	}

	// If the inner type is not a struct (map, slice, primitive, any/interface{}),
	// skip struct expansion and let BuildSchema handle it directly.
	probe := &StructField{TypeString: subTypeName}
	if strings.HasPrefix(subTypeName, "map[") || probe.IsPrimitive() || probe.IsAny() {
		f.TypeString = subTypeName
		builder.Fields = append(builder.Fields, f)
		return
	}

	// Handle array and pointer prefixes - keep them in originalTypeName but strip for type lookup
	arrayPrefix := ""
	if strings.HasPrefix(subTypeName, "[]") {
		arrayPrefix = "[]"
		subTypeName = strings.TrimPrefix(subTypeName, "[]")
	}
	if strings.HasPrefix(subTypeName, "*") {
		arrayPrefix = arrayPrefix + "*"
		subTypeName = strings.TrimPrefix(subTypeName, "*")
	}

	// After stripping array/pointer prefixes, check if bare type is primitive.
	// Catches StructField[[]string] where "string" should not be package-qualified.
	strippedProbe := &StructField{TypeString: subTypeName}
	if strippedProbe.IsPrimitive() || strippedProbe.IsAny() {
		f.TypeString = arrayPrefix + subTypeName
		builder.Fields = append(builder.Fields, f)
		return
	}

	// Store the original full type name with package path
	originalTypeName := subTypeName
	var subTypePackage string

	console.Logger.Debug("----Sub Type Name: %s (arrayPrefix: %s)\n", subTypeName, arrayPrefix)

	// Parse package and type name
	if strings.Contains(subTypeName, "/") {
		// Full package path like "github.com/griffnb/project/internal/models/billing_plan.FeatureSet"
		pathParts := strings.Split(subTypeName, "/")
		lastPart := pathParts[len(pathParts)-1]

		dotParts := strings.Split(lastPart, ".")
		if len(dotParts) < 2 {
			f.TypeString = subTypeName
			builder.Fields = append(builder.Fields, f)
			return
		}

		packageName := dotParts[0]
		typeName := dotParts[len(dotParts)-1]
		fullPackagePath := strings.Join(pathParts[:len(pathParts)-1], "/") + "/" + packageName
		// Preserve full import path so BuildSchema can propagate it for correct
		// cross-package resolution in buildSchemasRecursive. normalizeTypeName
		// will shorten it to "package.Type" form when building the $ref name.
		originalTypeName = arrayPrefix + fullPackagePath + "." + typeName

		subTypePackage = fullPackagePath
		subTypeName = typeName
	} else if strings.Contains(subTypeName, ".") {
		// Already in package.Type format
		subParts := strings.Split(subTypeName, ".")
		if len(subParts) < 2 {
			f.TypeString = subTypeName
			builder.Fields = append(builder.Fields, f)
			return
		}
		packageName := subParts[len(subParts)-2]
		typeName := subParts[len(subParts)-1]
		originalTypeName = arrayPrefix + fmt.Sprintf("%s.%s", packageName, typeName)

		subTypePackage = strings.Join(subParts[:len(subParts)-1], ".")
		subTypeName = typeName
	} else {
		// Same-package type — qualify with current package name
		if c.basePackage != nil {
			originalTypeName = arrayPrefix + c.basePackage.Name + "." + subTypeName
			subTypePackage = c.basePackage.PkgPath
		} else {
			f.TypeString = arrayPrefix + subTypeName
			builder.Fields = append(builder.Fields, f)
			return
		}
	}

	console.Logger.Debug("-----Final Sub type Package %s\n Final Sub Type Name: %s\n", subTypePackage, subTypeName)

	// Find the target package
	targetPkg := packageMap[subTypePackage]
	if targetPkg == nil {
		console.Logger.Debug("WARNING: Package not found in map for %s\n", subTypePackage)
		targetPkg = c.basePackage
	} else {
		console.Logger.Debug("-----Found target package: %s\n", targetPkg.PkgPath)
	}

	if targetPkg == nil {
		console.Logger.Debug("------No package available\n")
		f.TypeString = originalTypeName
		builder.Fields = append(builder.Fields, f)
		return
	}

	// Extract subfields
	console.Logger.Debug("\n\n-------Sub Package Struct-----: \n%s\n", subTypeName)
	subFields := c.ExtractFieldsRecursive(targetPkg, subTypeName, packageMap, make(map[string]bool))
	console.Logger.Debug("--------Extracted %d subfields for %s\n", len(subFields), subTypeName)

	for _, subField := range subFields {
		console.Logger.Debug("Sub Field: %s, Type: %s, Tag: %s\n", subField.Name, subField.Type, subField.Tag)
	}

	f.TypeString = originalTypeName
	f.Fields = subFields
	console.Logger.Debug("-------Set field %s with TypeString=%s and %d Fields\n", f.Name, f.TypeString, len(f.Fields))

	builder.Fields = append(builder.Fields, f)
}

func (c *CoreStructParser) ExtractFieldsRecursive(
	pkg *packages.Package,
	typeName string,
	_ map[string]*packages.Package,
	visited map[string]bool,
) []*StructField {
	// Create a unique cache key with package path
	cacheKey := pkg.PkgPath + ":" + typeName
	if visited[cacheKey] {
		return nil
	}
	visited[cacheKey] = true

	var fields []*StructField

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)

				if !ok || ts.Name.Name != typeName {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				console.Logger.Debug("----Matched StructType & Processing: %s (has %d fields)\n", ts.Name.Name, len(st.Fields.List))
				for i, field := range st.Fields.List {
					var fieldName string
					if len(field.Names) > 0 {
						fieldName = field.Names[0].Name
					} else {
						switch expr := field.Type.(type) {
						case *ast.Ident:
							fieldName = expr.Name
						case *ast.SelectorExpr:
							fieldName = expr.Sel.Name
						default:
							fieldName = "unknown"
						}
					}

					tag := ""
					if field.Tag != nil {
						tag = strings.Trim(field.Tag.Value, "`")
					}

					var fieldType types.Type
					var obj types.Object
					if len(field.Names) > 0 {
						if obj, ok = pkg.TypesInfo.Defs[field.Names[0]]; ok {
							fieldType = obj.Type()
						}
					} else {
						if typ := pkg.TypesInfo.Types[field.Type]; typ.Type != nil {
							fieldType = typ.Type
						}
					}

					console.Logger.Debug(
						"----[Field %d/%d] Validating Field Name: %s, Type: %s (%T), Tag: %s\n",
						i+1,
						len(st.Fields.List),
						fieldName,
						fieldType,
						fieldType,
						tag,
					)

					// Parse struct tags correctly: split by space first, then by colon
					tagMap := make(map[string]string)
					for _, part := range strings.Fields(tag) {
						kv := strings.SplitN(part, ":", 2)
						if len(kv) == 2 {
							tagMap[strings.Trim(kv[0], "`")] = strings.Trim(kv[1], `"`)
						}
					}

					jsonTag := tagMap["json"]
					columnTag := tagMap["column"]

					// Skip if json tag is explicitly "-"
					if jsonTag == "-" {
						console.Logger.Debug("Skipping field %s because json tag is '-'\n", fieldName)
						continue
					}

					// Skip if column tag is explicitly "-"
					if columnTag == "-" {
						console.Logger.Debug("Skipping field %s because column tag is '-'\n", fieldName)
						continue
					}

					// Handle embedded fields BEFORE tag checks
					// Embedded fields (no explicit name) need recursive expansion
					// regardless of their tags
					isEmbedded := len(field.Names) == 0
					if isEmbedded {
						if subFields, _, ok := c.checkNamed(fieldType); ok {
							if len(subFields) == 0 {
								console.Logger.Debug("Skipping empty embedded field: %s\n", fieldName)
								continue
							}
							fields = append(fields, subFields...)
							continue
						}
						// Embedded field that isn't a struct - skip
						console.Logger.Debug("Skipping non-struct embedded field: %s\n", fieldName)
						continue
					}

					// Skip if NEITHER json nor column tag exists (both are empty)
					if jsonTag == "" && columnTag == "" {
						console.Logger.Debug("Skipping field %s because it has no json or column tag\n", fieldName)
						continue
					}

					// Named struct fields (non-embedded)
					if subFields, _, ok := c.checkNamed(fieldType); ok {
						if len(subFields) == 0 {
							console.Logger.Debug("Skipping empty named field: %s\n", fieldName)
							continue
						}
						fields = append(fields, subFields...)
						continue
					}

					if subFields, typeName, ok := c.checkStruct(fieldType); ok {
						fields = append(fields, &StructField{
							Name:       fieldName,
							Type:       fieldType,
							Tag:        tag,
							TypeString: typeName,
							Fields:     subFields,
						})

						console.Logger.Debug("----Added Struct Field: %s of type %s with %d subfields\n", fieldName, typeName, len(subFields))
						continue
					}
					if subFields, typeName, ok := c.checkSlice(fieldType); ok {
						fields = append(fields, &StructField{
							Name:       fieldName,
							Type:       fieldType,
							Tag:        tag,
							TypeString: typeName,
							Fields:     subFields,
						})
						continue
					}

					if subFields, typeName, ok := c.checkMap(fieldType); ok {
						fields = append(fields, &StructField{
							Name:       fieldName,
							Type:       fieldType,
							Tag:        tag,
							TypeString: typeName,
							Fields:     subFields,
						})
						continue
					}

					fields = append(fields, &StructField{
						Name:       fieldName,
						Type:       fieldType,
						Tag:        tag,
						TypeString: fieldType.String(),
					})
				}
			}
		}
	}

	return fields
}

func (c *CoreStructParser) checkNamed(fieldType types.Type) ([]*StructField, *types.Named, bool) {
	named, ok := fieldType.(*types.Named)
	if ok {
		// Check if package is nil (can happen for built-in/universe types)
		pkg := named.Obj().Pkg()
		if pkg == nil {
			// Built-in type without package (like error interface) - skip
			return nil, nil, false
		}

		if strings.Contains(pkg.Path(), "/lib/model/fields") {
			return nil, nil, false
		}
		// Skip types that should be treated as primitives in Swagger
		tempField := &StructField{Type: fieldType}
		if tempField.IsSwaggerPrimitive() {
			return nil, nil, false
		}
		if _, ok := named.Underlying().(*types.Struct); ok {
			console.Logger.Debug("Found sub type Package %s Name %s\n", pkg.Path(), named.Obj().Name())
			nextPackage, ok := c.packageMap[pkg.Path()]
			if !ok {
				console.Logger.Debug("Package not found for %s\n", pkg.Path())
				return nil, nil, true
			}
			console.Logger.Debug("Next Package: %s\n", nextPackage.PkgPath)
			subFields := c.ExtractFieldsRecursive(nextPackage, named.Obj().Name(), c.packageMap, c.visited)
			return subFields, named, true
		}
	}

	return nil, nil, false
}

// getQualifiedTypeName returns the type name with package prefix if the type is from a different package
// e.g., if we're processing the "account" package and we find a type from "address" package,
// it returns "address.TypeName" instead of just "TypeName"
func getQualifiedTypeName(namedType *types.Named) string {
	if namedType == nil {
		return ""
	}
	pkg := namedType.Obj().Pkg()
	if pkg == nil {
		return namedType.Obj().Name()
	}
	// Use the package name (last segment of the path) as prefix
	pkgName := pkg.Name()
	return fmt.Sprintf("%s.%s", pkgName, namedType.Obj().Name())
}

func (c *CoreStructParser) checkStruct(fieldType types.Type) ([]*StructField, string, bool) {
	pointer, isPointer := fieldType.(*types.Pointer)
	if isPointer {
		fields, namedType, ok := c.checkNamed(pointer.Elem())
		if ok && namedType != nil {
			qualifiedName := getQualifiedTypeName(namedType)
			return fields, fmt.Sprintf("*%s", qualifiedName), true
		}
	} else {
		fields, namedType, ok := c.checkNamed(fieldType)
		if ok && namedType != nil {
			return fields, getQualifiedTypeName(namedType), true
		}
	}

	return nil, "", false
}

func (c *CoreStructParser) checkSlice(fieldType types.Type) ([]*StructField, string, bool) {
	slice, isSlice := fieldType.(*types.Slice)
	if isSlice {
		fields, structType, ok := c.checkStruct(slice.Elem())
		if ok {
			return fields, fmt.Sprintf("[]%s", structType), true
		}
	}

	return nil, "", false
}

func (c *CoreStructParser) checkMap(fieldType types.Type) ([]*StructField, string, bool) {
	mapType, isMap := fieldType.(*types.Map)
	if isMap {
		var mapPart string
		// TODO this is a weird hack that probably sholdnt exist
		if strings.Contains(fieldType.String(), "*github.com") {
			mapPart = strings.Split(fieldType.String(), "*github.com")[0]
		} else {
			mapPart = strings.Split(fieldType.String(), "github.com/")[0]
		}

		fields, sliceType, isSlice := c.checkSlice(mapType.Elem())
		if isSlice {
			return fields, fmt.Sprintf("%s%s", mapPart, sliceType), true
		}

		fields, structType, isStruct := c.checkStruct(mapType.Elem())
		if isStruct {
			return fields, fmt.Sprintf("%s%s", mapPart, structType), true
		}
	}

	return nil, "", false
}

// BuildAllSchemas generates both public and non-public schema variants for a type.
// packageNameOverride, when non-empty, is used as the definition prefix instead of
// deriving it from the last segment of pkgPath.  This is important when the Go
// package name differs from the import path segment (e.g. package "stripe" lives at
// path ".../stripe-go/v84").
// Returns a map of schema names to schemas (includes both base and Public variants).
func BuildAllSchemas(baseModule, pkgPath, typeName string, packageNameOverride ...string) (map[string]*spec.Schema, error) {
	parser := &CoreStructParser{}

	// Use override if provided, otherwise derive from pkgPath
	packageName := pkgPath
	if len(packageNameOverride) > 0 && packageNameOverride[0] != "" {
		packageName = packageNameOverride[0]
	} else if idx := strings.LastIndex(pkgPath, "/"); idx >= 0 {
		packageName = pkgPath[idx+1:]
	}

	// Lookup struct fields using existing LookupStructFields
	builder := parser.LookupStructFields(baseModule, pkgPath, typeName)
	if builder == nil {
		return nil, fmt.Errorf("failed to lookup struct fields for %s", typeName)
	}

	allSchemas := make(map[string]*spec.Schema)
	processed := make(map[string]bool) // Track processed types to avoid infinite recursion

	// Generate schemas for the main type with package prefix
	fullTypeName := packageName + "." + typeName
	err := buildSchemasRecursive(builder, typeName, false, allSchemas, processed, parser, baseModule, pkgPath, packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to build schemas for %s: %w", fullTypeName, err)
	}

	err = buildSchemasRecursive(builder, typeName+"Public", true, allSchemas, processed, parser, baseModule, pkgPath, packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to build public schemas for %s: %w", fullTypeName, err)
	}

	return allSchemas, nil
}

// buildSchemasRecursive recursively builds schemas for a type and all its nested types
func buildSchemasRecursive(
	builder *StructBuilder,
	schemaName string,
	public bool,
	allSchemas map[string]*spec.Schema,
	processed map[string]bool,
	parser *CoreStructParser,
	baseModule, pkgPath, packageName string,
) error {
	// Avoid infinite recursion
	if processed[schemaName] {
		return nil
	}
	processed[schemaName] = true

	// Extract base type name (remove Public suffix if present)
	baseTypeName := schemaName
	if public && strings.HasSuffix(schemaName, "Public") {
		baseTypeName = strings.TrimSuffix(schemaName, "Public")
	}

	// Build schema for current type
	// Create a parser-based enum lookup that can access the packages
	enumLookup := &ParserEnumLookup{Parser: parser, BaseModule: baseModule, PkgPath: pkgPath}
	// Force all fields to be required in response schemas
	schema, nestedTypes, err := builder.BuildSpecSchema(baseTypeName, public, true, enumLookup)
	if err != nil {
		return fmt.Errorf("failed to build schema for %s: %w", schemaName, err)
	}

	// Store the schema with package prefix
	fullSchemaName := packageName + "." + schemaName

	// Set title to create clean class names in code generators
	// Strategy:
	// 1. If package is a prefix of type (e.g., account.Account, account.AccountJoined)
	//    → use just the type name (Account, AccountJoined)
	// 2. Otherwise (e.g., account.Properties, billing_plan.FeatureSet)
	//    → combine as PascalCase (AccountProperties, BillingPlanFeatureSet)

	typeName := schemaName // Use schemaName to preserve "Public" suffix if present

	// Remove underscores/hyphens from package for comparison
	// e.g., billing_plan → billingplan
	packageNoSeparators := strings.ReplaceAll(strings.ReplaceAll(packageName, "_", ""), "-", "")

	if strings.HasPrefix(strings.ToLower(typeName), strings.ToLower(packageNoSeparators)) {
		// Package is a prefix of type name (case-insensitive, ignoring separators)
		// e.g., account.Account → Account, account.AccountJoined → AccountJoined
		//       billing_plan.BillingPlanJoined → BillingPlanJoined
		schema.Title = typeName
	} else {
		// Package and type don't align - combine them
		// e.g., account.Properties → AccountProperties
		// Convert package_name to PascalCase: billing_plan → BillingPlan
		packagePascal := toPascalCase(packageName)
		schema.Title = packagePascal + typeName
	}

	allSchemas[fullSchemaName] = schema

	// Also register under the canonical name if globalNameResolver produces a
	// different key. This handles NotUnique types where $ref uses the full-path
	// name (e.g., "github_com_chargebee_chargebee-go_v3_enum.Source") but the
	// definition was stored under the short key ("enum.Source").
	if globalNameResolver != nil && strings.Contains(pkgPath, "/") {
		lookupType := strings.TrimSuffix(schemaName, "Public")
		isPublicSchema := strings.HasSuffix(schemaName, "Public")
		canonicalName := globalNameResolver.ResolveDefinitionName(pkgPath + "." + lookupType)
		if isPublicSchema {
			canonicalName += "Public"
		}
		if canonicalName != fullSchemaName {
			allSchemas[canonicalName] = schema
		}
	}

	// Recursively process nested types
	for _, nestedTypeName := range nestedTypes {
		// Parse package name and type name from nested type.
		// Supports three forms:
		// 1. Full import path: "github.com/.../constants.RolePublic"
		// 2. Short qualified: "account.Properties", "billing_plan.FeatureSet"
		// 3. Unqualified: "Properties"
		var nestedPackageName, baseNestedType, nestedPkgPath string

		if strings.Contains(nestedTypeName, "/") {
			// Full import path — extract package path, package name, and type name
			lastDot := strings.LastIndex(nestedTypeName, ".")
			if lastDot >= 0 {
				nestedPkgPath = nestedTypeName[:lastDot]
				baseNestedType = nestedTypeName[lastDot+1:]
				if lastSlash := strings.LastIndex(nestedPkgPath, "/"); lastSlash >= 0 {
					nestedPackageName = nestedPkgPath[lastSlash+1:]
				} else {
					nestedPackageName = nestedPkgPath
				}
			}
		} else if strings.Contains(nestedTypeName, ".") {
			// Short qualified name — existing sibling-path logic
			parts := strings.Split(nestedTypeName, ".")
			nestedPackageName = parts[0]
			baseNestedType = parts[len(parts)-1]
			nestedPkgPath = pkgPath
			if nestedPackageName != packageName {
				if idx := strings.LastIndex(pkgPath, "/"); idx >= 0 {
					nestedPkgPath = pkgPath[:idx+1] + nestedPackageName
				} else {
					nestedPkgPath = nestedPackageName
				}
			}
		} else {
			// Unqualified — same package
			nestedPackageName = packageName
			baseNestedType = nestedTypeName
			nestedPkgPath = pkgPath
		}

		// Need to lookup the nested type's fields using the correct package path
		// Strip "Public" suffix for lookup — enum/non-struct types never have Public variants
		cleanNestedType := baseNestedType
		cleanNestedType = strings.TrimSuffix(cleanNestedType, "Public")

		nestedBuilder := parser.LookupStructFields(baseModule, nestedPkgPath, cleanNestedType)

		// Case 1: nil builder — external/unloadable type. Create opaque object definition.
		if nestedBuilder == nil {
			console.Logger.Debug(
				"$Yellow{Nested type %s in package %s has nil builder (external/unloadable), creating opaque object}\n",
				cleanNestedType,
				nestedPkgPath,
			)
			opaqueSchema := &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type:  []string{"object"},
					Title: toPascalCase(nestedPackageName) + cleanNestedType,
				},
			}
			opaqueKey := nestedPackageName + "." + cleanNestedType
			allSchemas[opaqueKey] = opaqueSchema

			processed[cleanNestedType] = true
			processed[cleanNestedType+"Public"] = true
			processed[nestedPackageName+"."+cleanNestedType] = true
			processed[nestedPackageName+"."+cleanNestedType+"Public"] = true
			continue
		}

		// Case 2 & 3: builder exists but has no fields — check if it's an enum or empty struct
		if len(nestedBuilder.Fields) == 0 {
			console.Logger.Debug(
				"$Yellow{Nested type %s is not a struct in package %s, checking if it's an enum}\n",
				cleanNestedType,
				nestedPkgPath,
			)

			// Try to get enum values for this type using the clean name (no Public suffix)
			enumLookup := &ParserEnumLookup{Parser: parser, BaseModule: baseModule, PkgPath: nestedPkgPath}
			cleanFullName := nestedPackageName + "." + cleanNestedType
			enums, err := enumLookup.GetEnumsForType(cleanFullName, nil)

			if err == nil && len(enums) > 0 {
				// Case 2: enum type — create enum definition
				console.Logger.Debug("$Green{Found enum type %s with %d values, creating enum definition}\n", cleanFullName, len(enums))

				baseEnumSchema := &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Title: toPascalCase(nestedPackageName) + cleanNestedType,
					},
				}

				switch enums[0].Value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
					baseEnumSchema.Type = []string{"integer"}
				case string:
					baseEnumSchema.Type = []string{"string"}
				case float32, float64:
					baseEnumSchema.Type = []string{"number"}
				default:
					baseEnumSchema.Type = []string{"integer"}
				}

				applyEnumsToSchema(baseEnumSchema, enums)

				baseSchemaKey := nestedPackageName + "." + cleanNestedType
				allSchemas[baseSchemaKey] = baseEnumSchema
				console.Logger.Debug("Created enum schema: %s\n", baseSchemaKey)

				// Also register enum under canonical name for NotUnique types
				if globalNameResolver != nil && strings.Contains(nestedPkgPath, "/") {
					canonicalName := globalNameResolver.ResolveDefinitionName(nestedPkgPath + "." + cleanNestedType)
					if canonicalName != baseSchemaKey {
						allSchemas[canonicalName] = baseEnumSchema
					}
				}
			} else {
				// Case 3: empty struct (no fields, not enum) — create opaque object definition
				console.Logger.Debug(
					"$Yellow{Nested type %s in package %s is empty struct, creating opaque object}\n",
					cleanNestedType,
					nestedPkgPath,
				)
				emptySchema := &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type:  []string{"object"},
						Title: toPascalCase(nestedPackageName) + cleanNestedType,
					},
				}
				emptyKey := nestedPackageName + "." + cleanNestedType
				allSchemas[emptyKey] = emptySchema
			}

			// Mark both base and Public as processed to prevent Public variant creation
			processed[cleanNestedType] = true
			processed[cleanNestedType+"Public"] = true
			processed[nestedPackageName+"."+cleanNestedType] = true
			processed[nestedPackageName+"."+cleanNestedType+"Public"] = true
			continue
		}

		// Generate both public and non-public variants for nested struct types
		err = buildSchemasRecursive(
			nestedBuilder,
			cleanNestedType,
			false,
			allSchemas,
			processed,
			parser,
			baseModule,
			nestedPkgPath,
			nestedPackageName,
		)
		if err != nil {
			return err
		}

		err = buildSchemasRecursive(
			nestedBuilder,
			cleanNestedType+"Public",
			true,
			allSchemas,
			processed,
			parser,
			baseModule,
			nestedPkgPath,
			nestedPackageName,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
