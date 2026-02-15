package structparser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

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

	structType := findStructType(t, file, "Account")
	service := NewService(nil, nil)

	schema, err := service.ParseStruct(file, structType.Fields)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Should merge embedded fields
	assert.Equal(t, 3, len(schema.Properties))
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "created_at")
	assert.Contains(t, schema.Properties, "name")
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
