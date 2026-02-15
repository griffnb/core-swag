package structparser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/griffnb/core-swag/internal/domain"
	"github.com/griffnb/core-swag/internal/registry"
	"github.com/stretchr/testify/require"
)

// TestRegistryLookupDebug helps debug why registry.FindTypeSpec() returns nil
// for embedded types in the integration test.
func TestRegistryLookupDebug(t *testing.T) {
	t.Log("=== Registry Lookup Debug Test ===")
	t.Log("This test reproduces the registry lookup issue with embedded structs")

	// Step 1: Create registry
	reg := registry.NewService()

	// Step 2: Parse test files that mimic the integration test structure
	// File 1: base/structure.go (the embedded type)
	baseFile := `package base

type Structure struct {
	ID        string ` + "`json:\"id\"`" + `
	CreatedAt int64  ` + "`json:\"created_at\"`" + `
}
`

	// File 2: account/account.go (embeds base.Structure)
	accountFile := `package account

import "testdata/base"

type DBColumns struct {
	base.Structure
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"last_name\"`" + `
}

type Account struct {
	DBColumns
	Email string ` + "`json:\"email\"`" + `
}
`

	// Parse files into AST
	fset := token.NewFileSet()

	baseAST, err := parser.ParseFile(fset, "testdata/base/structure.go", baseFile, parser.ParseComments)
	require.NoError(t, err)

	accountAST, err := parser.ParseFile(fset, "testdata/account/account.go", accountFile, parser.ParseComments)
	require.NoError(t, err)

	// Step 3: Collect files into registry
	err = reg.CollectAstFile(fset, "testdata/base", "testdata/base/structure.go", baseAST, domain.ParseAll)
	require.NoError(t, err)

	err = reg.CollectAstFile(fset, "testdata/account", "testdata/account/account.go", accountAST, domain.ParseAll)
	require.NoError(t, err)

	// Step 4: Parse types
	schemas, err := reg.ParseTypes()
	require.NoError(t, err)

	t.Logf("\n=== Registry parsed %d schemas ===", len(schemas))

	// Step 5: Debug - Print what's in the registry
	t.Log("\n=== Types in Registry (from schemas) ===")
	for typeDef := range schemas {
		t.Logf("  - %s (pkg: %s, file: %s)",
			typeDef.TypeName(),
			typeDef.PkgPath,
			typeDef.File.Name.Name)
	}

	uniqueDefs := reg.UniqueDefinitions()
	t.Log("\n=== Unique Definitions (lookup map) ===")
	for name, typeDef := range uniqueDefs {
		if typeDef != nil {
			t.Logf("  - %s → %s (pkg: %s)", name, typeDef.TypeName(), typeDef.PkgPath)
		} else {
			t.Logf("  - %s → nil", name)
		}
	}

	packages := reg.Packages()
	t.Log("\n=== Package Definitions ===")
	for pkgPath, pkgDef := range packages {
		t.Logf("Package: %s (name: %s)", pkgPath, pkgDef.Name)
		t.Logf("  Type Definitions:")
		for typeName, typeDef := range pkgDef.TypeDefinitions {
			if typeDef != nil {
				t.Logf("    - %s → %s", typeName, typeDef.TypeName())
			} else {
				t.Logf("    - %s → nil", typeName)
			}
		}
	}

	// Step 6: Try lookups that are failing in integration test
	t.Log("\n=== Lookup Tests ===")

	// Test 1: Look up "base.Structure" from account file context
	t.Log("\nLookup 1: 'base.Structure' from account file")
	result := reg.FindTypeSpec("base.Structure", accountAST)
	if result != nil {
		t.Logf("  ✓ Found: %s (pkg: %s)", result.TypeName(), result.PkgPath)
	} else {
		t.Logf("  ✗ NOT FOUND")
	}

	// Test 2: Look up "DBColumns" from account file context
	t.Log("\nLookup 2: 'DBColumns' from account file")
	result = reg.FindTypeSpec("DBColumns", accountAST)
	if result != nil {
		t.Logf("  ✓ Found: %s (pkg: %s)", result.TypeName(), result.PkgPath)
	} else {
		t.Logf("  ✗ NOT FOUND")
	}

	// Test 3: Look up "Structure" (simple name) from base file context
	t.Log("\nLookup 3: 'Structure' from base file")
	result = reg.FindTypeSpec("Structure", baseAST)
	if result != nil {
		t.Logf("  ✓ Found: %s (pkg: %s)", result.TypeName(), result.PkgPath)
	} else {
		t.Logf("  ✗ NOT FOUND")
	}

	// Test 4: Look up "Account" from account file context
	t.Log("\nLookup 4: 'Account' from account file")
	result = reg.FindTypeSpec("Account", accountAST)
	if result != nil {
		t.Logf("  ✓ Found: %s (pkg: %s)", result.TypeName(), result.PkgPath)
	} else {
		t.Logf("  ✗ NOT FOUND")
	}

	// Step 7: Check imports in account file
	t.Log("\n=== Imports in account file ===")
	for _, imp := range accountAST.Imports {
		path := ""
		if imp.Path != nil {
			path = imp.Path.Value
		}
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		}
		t.Logf("  - name: '%s', path: %s", name, path)
	}

	// Step 8: Key assertion - base.Structure should be found
	t.Log("\n=== Final Assertion ===")
	result = reg.FindTypeSpec("base.Structure", accountAST)
	if result == nil {
		t.Error("FAILED: registry.FindTypeSpec('base.Structure', accountAST) returned nil")
		t.Error("This is the root cause of empty properties in embedded structs")
	} else {
		t.Logf("SUCCESS: Found base.Structure: %s", result.TypeName())
	}
}

// TestRegistryLookupWithRealFiles uses the ACTUAL integration test files
// to debug the real problem.
func TestRegistryLookupWithRealFiles(t *testing.T) {
	t.Log("=== Registry Lookup with Real Files ===")

	reg := registry.NewService()
	fset := token.NewFileSet()

	// Parse actual test data files
	baseFile := "/Users/griffnb/projects/core-swag/testing/testdata/core_models/base/structure.go"
	accountFile := "/Users/griffnb/projects/core-swag/testing/testdata/core_models/account/account.go"

	baseAST, err := parser.ParseFile(fset, baseFile, nil, parser.ParseComments)
	require.NoError(t, err)

	accountAST, err := parser.ParseFile(fset, accountFile, nil, parser.ParseComments)
	require.NoError(t, err)

	// Collect into registry with proper package paths
	err = reg.CollectAstFile(fset, "github.com/griffnb/core-swag/testing/testdata/core_models/base",
		baseFile, baseAST, domain.ParseAll)
	require.NoError(t, err)

	err = reg.CollectAstFile(fset, "github.com/griffnb/core-swag/testing/testdata/core_models/account",
		accountFile, accountAST, domain.ParseAll)
	require.NoError(t, err)

	// Parse types
	schemas, err := reg.ParseTypes()
	require.NoError(t, err)

	t.Logf("\n=== Parsed %d schemas from real files ===", len(schemas))

	// Print what's in registry
	t.Log("\n=== Types from Real Files ===")
	for typeDef := range schemas {
		t.Logf("  - %s (pkg: %s)", typeDef.TypeName(), typeDef.PkgPath)
	}

	t.Log("\n=== Unique Definitions from Real Files ===")
	uniqueDefs := reg.UniqueDefinitions()
	for name, typeDef := range uniqueDefs {
		if typeDef != nil {
			t.Logf("  - %s → %s", name, typeDef.TypeName())
		}
	}

	t.Log("\n=== Package Definitions from Real Files ===")
	packages := reg.Packages()
	for pkgPath, pkgDef := range packages {
		t.Logf("Package: %s", pkgPath)
		t.Logf("  Name: %s", pkgDef.Name)
		t.Logf("  Type Definitions:")
		for typeName, typeDef := range pkgDef.TypeDefinitions {
			if typeDef != nil {
				t.Logf("    - %s → %s", typeName, typeDef.TypeName())
			}
		}
	}

	// Try lookups
	t.Log("\n=== Lookup Tests with Real Files ===")

	// The actual lookup that's failing in the integration test
	result := reg.FindTypeSpec("base.Structure", accountAST)
	t.Logf("Lookup 'base.Structure' from account.go: %v", result != nil)
	if result != nil {
		t.Logf("  Found: %s (pkg: %s)", result.TypeName(), result.PkgPath)
	}

	result = reg.FindTypeSpec("DBColumns", accountAST)
	t.Logf("Lookup 'DBColumns' from account.go: %v", result != nil)
	if result != nil {
		t.Logf("  Found: %s (pkg: %s)", result.TypeName(), result.PkgPath)
	}

	result = reg.FindTypeSpec("Structure", baseAST)
	t.Logf("Lookup 'Structure' from structure.go: %v", result != nil)
	if result != nil {
		t.Logf("  Found: %s (pkg: %s)", result.TypeName(), result.PkgPath)
	}

	// Imports from account file
	t.Log("\n=== Imports in Real account.go ===")
	for _, imp := range accountAST.Imports {
		path := ""
		if imp.Path != nil {
			path = imp.Path.Value
		}
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		}
		t.Logf("  - name: '%s', path: %s", name, path)
	}

	// Key assertion
	t.Log("\n=== Final Real File Assertion ===")
	result = reg.FindTypeSpec("base.Structure", accountAST)
	if result == nil {
		t.Error("FAILED: registry.FindTypeSpec('base.Structure', accountAST) returned nil with real files")
		t.Error("This is the ROOT CAUSE of the integration test failure")
	} else {
		t.Logf("SUCCESS: Found base.Structure with real files")
	}
}

// TestEmbeddedFieldResolution tests the full embedded field resolution flow
func TestEmbeddedFieldResolution(t *testing.T) {
	t.Log("=== Embedded Field Resolution Test ===")
	t.Log("This test simulates what StructParser.handleEmbeddedField() does")

	reg := registry.NewService()
	fset := token.NewFileSet()

	// Simple test case
	baseFile := `package base

type Structure struct {
	ID string ` + "`json:\"id\"`" + `
}
`

	accountFile := `package account

import "testdata/base"

type Account struct {
	base.Structure
	Email string ` + "`json:\"email\"`" + `
}
`

	baseAST, err := parser.ParseFile(fset, "testdata/base/structure.go", baseFile, parser.ParseComments)
	require.NoError(t, err)

	accountAST, err := parser.ParseFile(fset, "testdata/account/account.go", accountFile, parser.ParseComments)
	require.NoError(t, err)

	// Collect and parse
	err = reg.CollectAstFile(fset, "testdata/base", "testdata/base/structure.go", baseAST, domain.ParseAll)
	require.NoError(t, err)

	err = reg.CollectAstFile(fset, "testdata/account", "testdata/account/account.go", accountAST, domain.ParseAll)
	require.NoError(t, err)

	_, err = reg.ParseTypes()
	require.NoError(t, err)

	// Find the Account struct
	var accountStruct *ast.StructType
	for _, decl := range accountAST.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == "Account" {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							accountStruct = structType
							break
						}
					}
				}
			}
		}
	}

	require.NotNil(t, accountStruct, "Failed to find Account struct")
	require.NotNil(t, accountStruct.Fields, "Account has no fields")
	require.Greater(t, len(accountStruct.Fields.List), 0, "Account has no field list")

	// Get the first field (embedded base.Structure)
	embeddedField := accountStruct.Fields.List[0]
	require.Equal(t, 0, len(embeddedField.Names), "First field should be embedded (no names)")

	// Try to resolve the type name
	t.Log("\n=== Resolving Embedded Field ===")
	var typeName string
	switch expr := embeddedField.Type.(type) {
	case *ast.SelectorExpr:
		if pkg, ok := expr.X.(*ast.Ident); ok {
			typeName = pkg.Name + "." + expr.Sel.Name
			t.Logf("Resolved type name: %s", typeName)
		}
	default:
		t.Errorf("Unexpected embedded field type: %T", expr)
	}

	require.Equal(t, "base.Structure", typeName, "Should resolve to base.Structure")

	// Now try to look it up in the registry
	t.Log("\n=== Looking Up Embedded Type ===")
	typeDef := reg.FindTypeSpec(typeName, accountAST)

	if typeDef == nil {
		t.Error("FAILED: FindTypeSpec returned nil for embedded type")
		t.Error("This explains why embedded fields are skipped")

		// Debug: Show what we're looking for vs what's available
		t.Log("\n=== Debug Information ===")
		t.Logf("Looking for: '%s'", typeName)
		t.Log("Available in uniqueDefinitions:")
		for key := range reg.UniqueDefinitions() {
			t.Logf("  - '%s'", key)
		}
	} else {
		t.Logf("SUCCESS: Found type: %s", typeDef.TypeName())

		// Verify it's a struct
		if structType, ok := typeDef.TypeSpec.Type.(*ast.StructType); ok {
			t.Logf("Type is a struct with %d fields", len(structType.Fields.List))
		} else {
			t.Errorf("Type is not a struct: %T", typeDef.TypeSpec.Type)
		}
	}
}
