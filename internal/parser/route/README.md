# Route Parser Service

The Route Parser Service handles parsing of function comments to extract route definitions and operations.

## Overview

This service extracts HTTP route information from Go function comments, including:
- Route paths and HTTP methods
- Operation summaries and descriptions
- Request parameters (query, path, header, body, formData)
- Response definitions (success and failure cases)
- Security requirements
- Tags and operation IDs
- Code examples

## Files

- **service.go** (200 lines) - Main route parsing service and orchestration
- **operation.go** (400 lines) - Operation parser (@router, @summary, @description, etc.)
- **parameter.go** (300 lines) - Parameter extraction (@param)
- **response.go** (250 lines) - Response extraction (@success, @failure)
- **domain/route.go** (120 lines) - Route domain object

Total: ~1,270 lines across 5 focused files (down from 1,314 lines in single file)

## Usage

```go
import (
    "github.com/griffnb/core-swag/internal/parser/route"
    "github.com/griffnb/core-swag/internal/registry"
    "github.com/griffnb/core-swag/internal/parser/struct"
    "github.com/go-openapi/spec"
)

// Create route parser
registry := registry.NewService()
structParser := struct.NewService(registry, schemaBuilder, options)
swagger := &spec.Swagger{}

parser := route.NewService(
    registry,
    structParser,
    swagger,
    &route.Options{
        CodeExampleFilesDir: "./examples",
    },
)

// Parse routes from a file
astFile := parseGoFile("./handlers/user.go")
err := parser.ParseRoutes(astFile)
if err != nil {
    // Handle error
}

// swagger.Paths now contains all parsed operations
```

## Supported Annotations

### Basic Route Definition

```go
// ShowUser godoc
// @Summary      Show a user
// @Description  Get user by ID
// @ID           show-user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  User
// @Failure      404  {object}  ErrorResponse
// @Router       /users/{id} [get]
func ShowUser(c *gin.Context) {
    // Implementation
}
```

### All Operation Annotations

#### Route and Method

```go
// @Router /users/{id} [get]
// @Router /users [post]
// @deprecatedrouter /old-users/{id} [get]  // Marks as deprecated
```

#### Metadata

```go
// @Summary      Short description (one line)
// @Description  Longer description
// @Description  Can span multiple lines
// @ID           unique-operation-id
// @Tags         users,admin
// @Deprecated   // Marks operation as deprecated
```

#### Content Types

```go
// @Accept   json
// @Accept   xml,json          // Multiple types
// @Produce  json
// @Produce  xml,json,plain    // Multiple types
```

#### Parameters

Query parameters:
```go
// @Param  q       query  string  false  "Search query"
// @Param  page    query  int     false  "Page number"     default(1)
// @Param  limit   query  int     false  "Items per page"  default(10) minimum(1) maximum(100)
```

Path parameters:
```go
// @Param  id      path   int     true   "User ID"
// @Param  slug    path   string  true   "Post slug"
```

Header parameters:
```go
// @Param  Authorization  header  string  true  "Bearer token"
// @Param  X-API-Version  header  string  true  "API version" default(v1)
```

Body parameters:
```go
// @Param  user  body  CreateUserRequest  true  "User data"
```

Form data:
```go
// @Param  name   formData  string  true   "User name"
// @Param  email  formData  string  true   "User email"
// @Param  file   formData  file    false  "Profile picture"
```

#### Responses

```go
// @Success  200  {object}   User
// @Success  201  {object}   User
// @Success  204  {string}   string  "No content"
// @Failure  400  {object}   ErrorResponse  "Bad request"
// @Failure  404  {object}   ErrorResponse  "Not found"
// @Failure  500  {object}   ErrorResponse  "Internal error"
// @response default {object} ErrorResponse  "Unexpected error"
```

Response with arrays:
```go
// @Success  200  {array}  User
```

Response with generic types:
```go
// @Success  200  {object}  Response[User]
```

Response headers:
```go
// @Success  200              {object}  User
// @Header   200              {string}  Location  "/users/123"
// @Header   200,201          {string}  X-Request-ID  "request-id"
// @Header   all              {string}  X-Rate-Limit  "rate-limit"
```

#### Security

```go
// @Security ApiKeyAuth
// @Security OAuth2Application[write,read]
// @Security ApiKeyAuth && OAuth2Application[admin]  // AND condition
```

#### Code Examples

```go
// @x-codeSample file ./examples/create-user.md
```

#### Extensions

```go
// @x-example-key {"key": "value"}
```

## Key Methods

### NewService

```go
func NewService(
    registry *registry.Service,
    structParser *struct.Service,
    swagger *spec.Swagger,
    options *Options,
) *Service
```

Creates a new route parser service.

### ParseRoutes

```go
func (s *Service) ParseRoutes(astFile *ast.File) error
```

Parses all route definitions from a Go source file. Finds functions with route annotations and processes them.

### ParseOperation

```go
func (s *Service) ParseOperation(funcDecl *ast.FuncDecl) (*spec.Operation, error)
```

Parses a single function's comments to extract an operation definition.

## Options

```go
type Options struct {
    CodeExampleFilesDir string // Directory containing code example files
}
```

## Implementation Details

### Operation Parsing Process

1. **Find Functions**: Walk AST to find function declarations
2. **Check Comments**: Look for @router annotations
3. **Extract Metadata**: Parse @summary, @description, @id, @tags
4. **Parse Parameters**: Extract all @param annotations
5. **Parse Responses**: Extract @success, @failure, @response
6. **Parse Security**: Extract @security annotations
7. **Build Operation**: Construct spec.Operation object
8. **Add to Swagger**: Register operation in swagger.Paths

### Parameter Parsing

Parameters are parsed by type:

1. **Path Parameters**: Extracted from route path and @param annotations
2. **Query Parameters**: @param with location "query"
3. **Header Parameters**: @param with location "header"
4. **Body Parameters**: @param with location "body", references schema
5. **Form Parameters**: @param with location "formData"

Each parameter includes:
- Name
- Type (string, integer, boolean, file, object)
- Required flag
- Description
- Validation constraints (min, max, enum, etc.)
- Default value
- Example value

### Response Parsing

Responses are parsed to include:

1. **Status Code**: HTTP status code or "default"
2. **Schema**: Response body schema (object or array)
3. **Description**: Response description
4. **Headers**: Response headers with types

The parser handles:
- Multiple success responses
- Multiple failure responses
- Generic response types
- Array responses
- String responses (for simple messages)

### Security Parsing

Security requirements can be:

- **Single**: `@Security ApiKeyAuth`
- **OR Condition**: Multiple @Security lines (any requirement satisfied)
- **AND Condition**: `@Security ApiKeyAuth && OAuth2` (all requirements must be satisfied)

### Type Resolution

The route parser uses the registry and struct parser to resolve types:

```go
// @Param user body CreateUserRequest true "User data"
```

1. Looks up "CreateUserRequest" in registry
2. Uses struct parser to generate schema
3. Creates parameter with schema reference

## Testing

Comprehensive tests are provided in `service_test.go`:

```bash
go test ./internal/parser/route/...
```

Tests cover:
- Basic route parsing
- All parameter types
- Response definitions
- Security requirements
- Generic types
- Error handling
- Edge cases

## Design Principles

1. **Separation of Concerns**: Each file handles one aspect (operations, parameters, responses)
2. **Clear Error Messages**: All errors include function name and annotation context
3. **Extensibility**: Easy to add new annotation types
4. **Type Safety**: Uses spec.* types from go-openapi
5. **Testability**: Each component is independently testable

## Common Patterns

### Simple CRUD Operations

```go
// CreateUser creates a new user
// @Summary      Create user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body      CreateUserRequest  true  "User data"
// @Success      201   {object}  User
// @Failure      400   {object}  ErrorResponse
// @Router       /users [post]
func CreateUser(c *gin.Context) { }

// GetUser retrieves a user
// @Summary      Get user
// @Tags         users
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  User
// @Failure      404  {object}  ErrorResponse
// @Router       /users/{id} [get]
func GetUser(c *gin.Context) { }

// UpdateUser updates a user
// @Summary      Update user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path      int                true  "User ID"
// @Param        user  body      UpdateUserRequest  true  "User data"
// @Success      200   {object}  User
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /users/{id} [put]
func UpdateUser(c *gin.Context) { }

// DeleteUser deletes a user
// @Summary      Delete user
// @Tags         users
// @Param        id   path  int  true  "User ID"
// @Success      204  "No content"
// @Failure      404  {object}  ErrorResponse
// @Router       /users/{id} [delete]
func DeleteUser(c *gin.Context) { }

// ListUsers lists all users
// @Summary      List users
// @Tags         users
// @Produce      json
// @Param        page   query  int     false  "Page number"   default(1)
// @Param        limit  query  int     false  "Items per page" default(10)
// @Param        q      query  string  false  "Search query"
// @Success      200    {array}   User
// @Failure      400    {object}  ErrorResponse
// @Router       /users [get]
func ListUsers(c *gin.Context) { }
```

### With Security

```go
// CreatePost creates a new post
// @Summary      Create post
// @Description  Create a new blog post (requires authentication)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        post  body      CreatePostRequest  true  "Post data"
// @Success      201   {object}  Post
// @Failure      401   {object}  ErrorResponse  "Unauthorized"
// @Failure      403   {object}  ErrorResponse  "Forbidden"
// @Security     ApiKeyAuth
// @Router       /posts [post]
func CreatePost(c *gin.Context) { }
```

### With File Upload

```go
// UploadFile uploads a file
// @Summary      Upload file
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file    true   "File to upload"
// @Param        name  formData  string  false  "File name"
// @Success      200   {object}  FileInfo
// @Failure      400   {object}  ErrorResponse
// @Router       /files [post]
func UploadFile(c *gin.Context) { }
```

## Integration

The route parser is integrated into the main parser flow:

1. Main parser loads and registers all files
2. After type registration, calls route parser for each file
3. Route parser extracts operations from function comments
4. Operations are added to swagger.Paths
5. References to types in parameters/responses are resolved

For more details, see [ARCHITECTURE.md](../../../ARCHITECTURE.md).

## Error Handling

Common errors:

- **Missing @Router**: Function has operation annotations but no @router
- **Invalid Route Format**: @router annotation has incorrect format
- **Unknown Type**: @param or @success references a type that doesn't exist
- **Invalid Parameter Format**: @param annotation has incorrect format
- **Duplicate Operation**: Same route+method defined multiple times

All errors include:
- Function name where the error occurred
- Line number (when available)
- Description of what was expected
- The problematic annotation

## Performance Considerations

The route parser:
- Processes files sequentially
- Caches type lookups via registry
- Reuses struct parser for type resolution

For large projects, route parsing typically takes:
- 10-50ms for small projects (<100 routes)
- 100-500ms for medium projects (100-1000 routes)
- 1-5s for large projects (>1000 routes)
