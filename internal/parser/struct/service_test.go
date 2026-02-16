package structparser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/griffnb/core-swag/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseStruct_SimpleStruct tests parsing a struct with no special tags
func TestParseStruct_SimpleStruct(t *testing.T) {
	source := `
package testpkg

// Account represents a user account
type Account struct {
	ID   int64  ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`
	// Parse source into AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	// Find the struct type
	structType := findStructType(t, file, "Account")

	// Create service
	service := NewService(nil, nil)

	// Parse the struct
	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Verify schema properties
	assert.Contains(t, schema.Type, "object")
	assert.Equal(t, 2, len(schema.Properties))
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "name")

	// Verify field types
	assert.Contains(t, schema.Properties["id"].Type, "integer")
	assert.Contains(t, schema.Properties["name"].Type, "string")
}

// TestParseStruct_WithPublicFields tests struct with public:"view" and public:"edit" tags
func TestParseStruct_WithPublicFields(t *testing.T) {
	source := `
package testpkg

type Account struct {
	ID       int64  ` + "`json:\"id\" public:\"view\"`" + `
	Name     string ` + "`json:\"name\" public:\"view\"`" + `
	Email    string ` + "`json:\"email\" public:\"edit\"`" + `
	Password string ` + "`json:\"password\"`" + ` // private field
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	// This test verifies that ParseStruct correctly identifies public fields
	// The actual "Public" variant generation will be tested separately
	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Base schema should have all fields
	assert.Equal(t, 4, len(schema.Properties))
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "name")
	assert.Contains(t, schema.Properties, "email")
	assert.Contains(t, schema.Properties, "password")
}

// TestParseStruct_CustomModel tests fields.StructField[T] custom model pattern
func TestParseStruct_CustomModel(t *testing.T) {
	source := `
package testpkg

import "github.com/example/fields"

type Account struct {
	FirstName fields.StructField[string] ` + "`json:\"first_name\" public:\"edit\"`" + `
	Age       fields.StructField[int64]  ` + "`json:\"age\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Should extract inner types: string and int64
	assert.Equal(t, 2, len(schema.Properties))
	assert.Contains(t, schema.Properties, "first_name")
	assert.Contains(t, schema.Properties, "age")

	// Verify extracted types (not the wrapper)
	assert.Contains(t, schema.Properties["first_name"].Type, "string")
	assert.Contains(t, schema.Properties["age"].Type, "integer")
}

// TestParseStruct_EmbeddedStruct tests embedded struct field merging
func TestParseStruct_EmbeddedStruct(t *testing.T) {
	t.Run("simple embedding same package", func(t *testing.T) {
		source := `
package testpkg

type Inner struct {
	Field string ` + "`json:\"field\"`" + `
}

type Outer struct {
	Inner
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		// Create registry and register types
		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Outer")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Outer should have "field" property from Inner
		assert.Equal(t, 1, len(schema.Properties), "Should have 1 property from embedded Inner")
		assert.Contains(t, schema.Properties, "field")
		assert.Contains(t, schema.Properties["field"].Type, "string")
	})

	t.Run("pointer embedding", func(t *testing.T) {
		source := `
package testpkg

type Inner struct {
	Field string ` + "`json:\"field\"`" + `
}

type Outer struct {
	*Inner
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Outer")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Should strip pointer and merge properties
		assert.Equal(t, 1, len(schema.Properties))
		assert.Contains(t, schema.Properties, "field")
	})

	t.Run("chained embeddings", func(t *testing.T) {
		source := `
package testpkg

type Level3 struct {
	Field string ` + "`json:\"field\"`" + `
}

type Level2 struct {
	Level3
}

type Level1 struct {
	Level2
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Level1")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Level1 should have "field" from Level3 through Level2
		assert.Equal(t, 1, len(schema.Properties), "Should have field from Level3")
		assert.Contains(t, schema.Properties, "field")
	})

	t.Run("multiple embeddings", func(t *testing.T) {
		source := `
package testpkg

type Inner1 struct {
	Field1 string ` + "`json:\"field1\"`" + `
}

type Inner2 struct {
	Field2 string ` + "`json:\"field2\"`" + `
}

type Outer struct {
	Inner1
	Inner2
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Outer")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Should have properties from both Inner1 and Inner2
		assert.Equal(t, 2, len(schema.Properties))
		assert.Contains(t, schema.Properties, "field1")
		assert.Contains(t, schema.Properties, "field2")
	})

	t.Run("embedded with json tag - not truly embedded", func(t *testing.T) {
		source := `
package testpkg

type Inner struct {
	Field string ` + "`json:\"field\"`" + `
}

type Outer struct {
	Inner ` + "`json:\"inner\"`" + `
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Outer")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Should NOT merge - has json tag so treated as named field
		assert.Equal(t, 1, len(schema.Properties))
		assert.Contains(t, schema.Properties, "inner")
		// "inner" should be object type, not merged
		assert.Contains(t, schema.Properties["inner"].Type, "object")
	})

	t.Run("empty embedded struct", func(t *testing.T) {
		source := `
package testpkg

type Empty struct {
}

type Outer struct {
	Empty
	Name string ` + "`json:\"name\"`" + `
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Outer")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Should skip empty embedded, only have Name
		assert.Equal(t, 1, len(schema.Properties))
		assert.Contains(t, schema.Properties, "name")
	})

	t.Run("mixed embedded and direct fields", func(t *testing.T) {
		source := `
package testpkg

type Inner struct {
	InnerField string ` + "`json:\"inner_field\"`" + `
}

type Outer struct {
	Inner
	DirectField string ` + "`json:\"direct_field\"`" + `
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Outer")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Should have both embedded and direct fields
		assert.Equal(t, 2, len(schema.Properties))
		assert.Contains(t, schema.Properties, "inner_field")
		assert.Contains(t, schema.Properties, "direct_field")
	})

	t.Run("embedded with BaseModel pattern", func(t *testing.T) {
		source := `
package testpkg

type BaseModel struct {
	ID        int64  ` + "`json:\"id\"`" + `
	CreatedAt string ` + "`json:\"created_at\"`" + `
}

type Account struct {
	BaseModel
	Name string ` + "`json:\"name\"`" + `
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		reg := setupTestRegistry(t, source)

		structType := findStructType(t, file, "Account")
		service := NewService(reg, nil)

		schema, err := service.ParseStruct(file, structType.Fields)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Should merge embedded BaseModel fields
		assert.Equal(t, 3, len(schema.Properties))
		assert.Contains(t, schema.Properties, "id")
		assert.Contains(t, schema.Properties, "created_at")
		assert.Contains(t, schema.Properties, "name")
	})
}

// TestParseStruct_PointerFields tests pointer types (*string, *int64)
func TestParseStruct_PointerFields(t *testing.T) {
	source := `
package testpkg

type Account struct {
	Name        *string ` + "`json:\"name\"`" + `
	Age         *int64  ` + "`json:\"age\"`" + `
	IsActive    *bool   ` + "`json:\"is_active\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Pointer fields should have correct underlying types
	assert.Contains(t, schema.Properties["name"].Type, "string")
	assert.Contains(t, schema.Properties["age"].Type, "integer")
	assert.Contains(t, schema.Properties["is_active"].Type, "boolean")
}

// TestParseStruct_SliceFields tests slice field types
func TestParseStruct_SliceFields(t *testing.T) {
	source := `
package testpkg

type Account struct {
	Tags      []string ` + "`json:\"tags\"`" + `
	Counts    []int64  ` + "`json:\"counts\"`" + `
	Flags     []bool   ` + "`json:\"flags\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// All should be array types
	assert.Contains(t, schema.Properties["tags"].Type, "array")
	assert.Contains(t, schema.Properties["counts"].Type, "array")
	assert.Contains(t, schema.Properties["flags"].Type, "array")

	// Verify item types
	assert.NotNil(t, schema.Properties["tags"].Items)
	assert.NotNil(t, schema.Properties["counts"].Items)
	assert.NotNil(t, schema.Properties["flags"].Items)
}

// TestParseStruct_MapFields tests map field types
func TestParseStruct_MapFields(t *testing.T) {
	source := `
package testpkg

type Account struct {
	Metadata map[string]string ` + "`json:\"metadata\"`" + `
	Scores   map[string]int64  ` + "`json:\"scores\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Maps are objects with additionalProperties
	assert.Contains(t, schema.Properties["metadata"].Type, "object")
	assert.Contains(t, schema.Properties["scores"].Type, "object")

	// Should have additionalProperties defined
	assert.NotNil(t, schema.Properties["metadata"].AdditionalProperties)
	assert.NotNil(t, schema.Properties["scores"].AdditionalProperties)
}

// TestParseStruct_EmptyStruct tests struct with no fields
func TestParseStruct_EmptyStruct(t *testing.T) {
	source := `
package testpkg

type Empty struct {
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Empty")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Should be valid object with no properties
	assert.Contains(t, schema.Type, "object")
	assert.Equal(t, 0, len(schema.Properties))
}

// TestParseStruct_IgnoredFields tests json:"-" and swaggerignore:"true"
func TestParseStruct_IgnoredFields(t *testing.T) {
	source := `
package testpkg

type Account struct {
	ID       int64  ` + "`json:\"id\"`" + `
	Internal string ` + "`json:\"-\"`" + `
	Secret   string ` + "`json:\"secret\" swaggerignore:\"true\"`" + `
	Name     string ` + "`json:\"name\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Should only have visible fields
	assert.Equal(t, 2, len(schema.Properties))
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "name")
	assert.NotContains(t, schema.Properties, "Internal")
	assert.NotContains(t, schema.Properties, "secret")
}

// TestParseStruct_OmitEmpty tests omitempty tag handling
func TestParseStruct_OmitEmpty(t *testing.T) {
	source := `
package testpkg

type Account struct {
	ID       int64  ` + "`json:\"id\"`" + `
	Name     string ` + "`json:\"name,omitempty\"`" + `
	Email    string ` + "`json:\"email,omitempty\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// All fields should be present
	assert.Equal(t, 3, len(schema.Properties))

	// Fields with omitempty should not be in required list
	// (This test verifies the schema structure, required handling tested separately)
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "name")
	assert.Contains(t, schema.Properties, "email")
}

// TestParseStruct_ValidationTags tests validation tag extraction
func TestParseStruct_ValidationTags(t *testing.T) {
	source := `
package testpkg

type Account struct {
	Name  string ` + "`json:\"name\" validate:\"required,min=3,max=50\"`" + `
	Age   int64  ` + "`json:\"age\" validate:\"required,gte=18,lte=120\"`" + `
	Email string ` + "`json:\"email\" binding:\"required\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Verify required fields are marked
	assert.Equal(t, 3, len(schema.Required))
	assert.Contains(t, schema.Required, "name")
	assert.Contains(t, schema.Required, "age")
	assert.Contains(t, schema.Required, "email")

	// Verify min/max constraints
	nameSchema := schema.Properties["name"]
	assert.NotNil(t, nameSchema.MinLength)
	assert.Equal(t, int64(3), *nameSchema.MinLength)
	assert.NotNil(t, nameSchema.MaxLength)
	assert.Equal(t, int64(50), *nameSchema.MaxLength)

	ageSchema := schema.Properties["age"]
	assert.NotNil(t, ageSchema.Minimum)
	assert.Equal(t, float64(18), *ageSchema.Minimum)
	assert.NotNil(t, ageSchema.Maximum)
	assert.Equal(t, float64(120), *ageSchema.Maximum)
}

// TestParseStruct_PackageQualifiedTypes tests types with package qualifiers
func TestParseStruct_PackageQualifiedTypes(t *testing.T) {
	source := `
package testpkg

import (
	"time"
	"github.com/google/uuid"
)

type Account struct {
	ID        uuid.UUID ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// UUID and Time should be string types in OpenAPI
	assert.Contains(t, schema.Properties["id"].Type, "string")
	assert.Contains(t, schema.Properties["created_at"].Type, "string")
}

// TestParseStruct_ArrayOfPointers tests []*Type pattern
func TestParseStruct_ArrayOfPointers(t *testing.T) {
	source := `
package testpkg

type User struct {
	Name string ` + "`json:\"name\"`" + `
}

type Account struct {
	Owners []*User ` + "`json:\"owners\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Should be array type
	assert.Contains(t, schema.Properties["owners"].Type, "array")
	assert.NotNil(t, schema.Properties["owners"].Items)
}

// TestParseField_SimpleField tests parsing a single field
func TestParseField_SimpleField(t *testing.T) {
	source := `
package testpkg

type Account struct {
	Name string ` + "`json:\"name\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	field := structType.Fields.List[0]

	service := NewService(nil, nil)

	properties, required, err := service.ParseField(file, field)
	require.NoError(t, err)
	require.NotNil(t, properties)

	// Should have one property
	assert.Equal(t, 1, len(properties))
	assert.Contains(t, properties, "name")
	assert.Contains(t, properties["name"].Type, "string")

	// Not marked as required (no validation tags)
	assert.Equal(t, 0, len(required))
}

// TestParseField_WithValidation tests field with validation tags
func TestParseField_WithValidation(t *testing.T) {
	source := `
package testpkg

type Account struct {
	Name string ` + "`json:\"name\" validate:\"required,min=3\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	field := structType.Fields.List[0]

	service := NewService(nil, nil)

	properties, required, err := service.ParseField(file, field)
	require.NoError(t, err)

	// Should be marked as required
	assert.Equal(t, 1, len(required))
	assert.Contains(t, required, "name")

	// Should have min constraint
	assert.NotNil(t, properties["name"].MinLength)
	assert.Equal(t, int64(3), *properties["name"].MinLength)
}

// TestShouldGeneratePublic_AllPrivate tests no public variant for all private fields
func TestShouldGeneratePublic_AllPrivate(t *testing.T) {
	source := `
package testpkg

type Account struct {
	ID       int64  ` + "`json:\"id\"`" + `
	Password string ` + "`json:\"password\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	// Check if public variant should be generated
	shouldGenerate := service.ShouldGeneratePublic(structType.Fields)
	assert.False(t, shouldGenerate, "Should not generate public variant when all fields are private")
}

// TestShouldGeneratePublic_WithPublicFields tests public variant generation
func TestShouldGeneratePublic_WithPublicFields(t *testing.T) {
	source := `
package testpkg

type Account struct {
	ID       int64  ` + "`json:\"id\" public:\"view\"`" + `
	Password string ` + "`json:\"password\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	// Should detect public fields
	shouldGenerate := service.ShouldGeneratePublic(structType.Fields)
	assert.True(t, shouldGenerate, "Should generate public variant when public fields exist")
}

// TestBuildPublicSchema tests creating Public variant with only public fields
func TestBuildPublicSchema_Subset(t *testing.T) {
	source := `
package testpkg

type Account struct {
	ID       int64  ` + "`json:\"id\" public:\"view\"`" + `
	Name     string ` + "`json:\"name\" public:\"view\"`" + `
	Email    string ` + "`json:\"email\" public:\"edit\"`" + `
	Password string ` + "`json:\"password\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	baseSchema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)

	// Build public variant
	publicSchema, err := service.BuildPublicSchema(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, publicSchema)

	// Public schema should only have public fields
	assert.Equal(t, 3, len(publicSchema.Properties))
	assert.Contains(t, publicSchema.Properties, "id")
	assert.Contains(t, publicSchema.Properties, "name")
	assert.Contains(t, publicSchema.Properties, "email")
	assert.NotContains(t, publicSchema.Properties, "password")

	// Base schema should still have all fields
	assert.Equal(t, 4, len(baseSchema.Properties))
}

// Helper function to find struct type in AST
func findStructType(t *testing.T, file *ast.File, name string) *ast.StructType {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != name {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			require.True(t, ok, "Type %s is not a struct", name)
			return structType
		}
	}
	t.Fatalf("Struct %s not found in AST", name)
	return nil
}

// Helper function to set up test registry with parsed types
func setupTestRegistry(t *testing.T, source string) *registry.Service {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	reg := registry.NewService()

	// Collect the file into registry
	err = reg.CollectAstFile(fset, "testpkg", "test.go", file, 0)
	require.NoError(t, err)

	// Parse types to register them
	_, err = reg.ParseTypes()
	require.NoError(t, err)

	return reg
}

// TestNoPublicAnnotation tests that @NoPublic annotation prevents Public variant generation
func TestNoPublicAnnotation(t *testing.T) {
	source := `
package testpkg

// ErrorResponse represents an error response
// @NoPublic
type ErrorResponse struct {
	Success bool   ` + "`json:\"success\" public:\"view\"`" + `
	Error   string ` + "`json:\"error\" public:\"view\"`" + `
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool ` + "`json:\"success\" public:\"view\"`" + `
	Data    any  ` + "`json:\"data\" public:\"view\"`" + `
}
`

	// Parse source into AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	// Create service
	service := NewService(nil, nil)

	// Find ErrorResponse genDecl, typeSpec, and struct
	errorResponseGenDecl, errorResponseTypeSpec := findGenDeclAndTypeSpec(t, file, "ErrorResponse")
	errorResponseStruct := findStructType(t, file, "ErrorResponse")

	// ErrorResponse has @NoPublic, so shouldGeneratePublicInternal should return false
	shouldGenerate := service.shouldGeneratePublicInternal(file, errorResponseGenDecl, errorResponseTypeSpec, errorResponseStruct.Fields)
	assert.False(t, shouldGenerate, "ErrorResponse with @NoPublic should not generate Public variant")

	// Find SuccessResponse genDecl, typeSpec, and struct
	successResponseGenDecl, successResponseTypeSpec := findGenDeclAndTypeSpec(t, file, "SuccessResponse")
	successResponseStruct := findStructType(t, file, "SuccessResponse")

	// SuccessResponse does NOT have @NoPublic, so shouldGeneratePublicInternal should return true
	shouldGenerate = service.shouldGeneratePublicInternal(file, successResponseGenDecl, successResponseTypeSpec, successResponseStruct.Fields)
	assert.True(t, shouldGenerate, "SuccessResponse without @NoPublic should generate Public variant")
}

// Helper to find GenDecl and TypeSpec by name
func findGenDeclAndTypeSpec(t *testing.T, file *ast.File, name string) (*ast.GenDecl, *ast.TypeSpec) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typeSpec.Name.Name == name {
				return genDecl, typeSpec
			}
		}
	}
	t.Fatalf("TypeSpec %s not found", name)
	return nil, nil
}

// Helper to find TypeSpec by name
func findTypeSpec(t *testing.T, file *ast.File, name string) *ast.TypeSpec {
	_, typeSpec := findGenDeclAndTypeSpec(t, file, name)
	return typeSpec
}
