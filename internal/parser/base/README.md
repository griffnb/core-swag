# Base Parser Service

The Base Parser Service handles parsing of general API information from Go comments.

## Overview

This service extracts high-level API metadata from comments in the main Go file, including:
- API title, version, and description
- Terms of service
- Contact information
- License information
- Host and base path
- Security definitions
- External documentation references

## Files

- **service.go** (170 lines) - Main service and orchestration
- **info.go** (75 lines) - General API info parsing (@title, @version, @description)
- **security.go** (130 lines) - Security definitions parser
- **extensions.go** (60 lines) - Extension handling (x-* fields)
- **helpers.go** (75 lines) - Utility functions

Total: ~510 lines across 5 focused files

## Usage

```go
import (
    "github.com/griffnb/core-swag/internal/parser/base"
    "github.com/go-openapi/spec"
)

// Create a new base parser
swagger := &spec.Swagger{}
parser := base.NewService(swagger)

// Parse general info from main file
err := parser.ParseGeneralInfo(mainFilePath, mainPackage)
if err != nil {
    // Handle error
}

// swagger.Info, swagger.SecurityDefinitions, etc. are now populated
```

## Supported Annotations

### General API Info

```go
// @title           Swagger Example API
// @version         1.0
// @description     This is a sample server.
// @description     It demonstrates multi-line descriptions.
//
// @termsOfService  http://swagger.io/terms/
//
// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io
//
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
//
// @host      localhost:8080
// @BasePath  /api/v1
//
// @schemes   http https
//
// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
```

### Security Definitions

```go
// @securityDefinitions.basic BasicAuth

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// @securitydefinitions.oauth2.application OAuth2Application
// @tokenUrl https://example.com/oauth/token
// @scope.write Grants write access
// @scope.admin Grants read and write access

// @securitydefinitions.oauth2.implicit OAuth2Implicit
// @authorizationurl https://example.com/oauth/authorize
// @scope.read Grants read access

// @securitydefinitions.oauth2.password OAuth2Password
// @tokenUrl https://example.com/oauth/token
// @scope.write Grants write access

// @securitydefinitions.oauth2.accessCode OAuth2AccessCode
// @tokenUrl https://example.com/oauth/token
// @authorizationurl https://example.com/oauth/authorize
// @scope.admin Grants read and write access
```

### Extensions

```go
// @x-example-key {"key": "value"}
```

## Key Methods

### NewService

```go
func NewService(swagger *spec.Swagger) *Service
```

Creates a new base parser service.

### ParseGeneralInfo

```go
func (s *Service) ParseGeneralInfo(mainAPIFile string, mainPkg *ast.Package) error
```

Parses general API information from the main file's comments. This is the primary entry point.

## Implementation Details

### Info Parsing

The service extracts API metadata by:
1. Finding package-level comments in the main file
2. Parsing each comment line for recognized annotations
3. Populating the swagger.Info struct with extracted data
4. Supporting multi-line values for description and other fields

### Security Definitions

Security definitions are parsed in a multi-step process:
1. Detect security definition type from annotation
2. Parse type-specific attributes (tokenUrl, authorizationUrl, scopes)
3. Create appropriate security scheme objects
4. Register schemes in swagger.SecurityDefinitions

### Extension Handling

Extensions (fields starting with `x-`) are:
1. Parsed as JSON values
2. Validated for correct format
3. Added to the appropriate swagger objects

## Testing

Comprehensive tests are provided in `service_test.go`:

```bash
go test ./internal/parser/base/...
```

Tests cover:
- Basic API info parsing
- Multi-line descriptions
- Security definitions (all types)
- Extension parsing
- Error handling

## Design Principles

1. **Single Responsibility**: Only parses general API info, not routes or types
2. **Stateless**: Operates on provided swagger object without maintaining state
3. **Clear File Organization**: Each file handles a specific aspect of parsing
4. **Error Handling**: All errors are wrapped with context
5. **Extensibility**: Easy to add new annotation types

## Common Use Cases

### Minimal Setup

```go
// @title My API
// @version 1.0
// @host localhost:8080
// @BasePath /api/v1
```

### With Security

```go
// @title My API
// @version 1.0
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @host localhost:8080
// @BasePath /api/v1
```

### Full Configuration

```go
// @title My API
// @version 1.0
// @description This is a comprehensive API.
// @description With multiple description lines.
//
// @termsOfService http://example.com/terms/
//
// @contact.name API Support
// @contact.url http://example.com/support
// @contact.email support@example.com
//
// @license.name MIT
// @license.url http://opensource.org/licenses/MIT
//
// @host api.example.com
// @BasePath /v1
// @schemes https
//
// @securityDefinitions.oauth2.application OAuth2
// @tokenUrl https://example.com/oauth/token
// @scope.read Grants read access
// @scope.write Grants write access
```

## Integration

The base parser is integrated into the main parser flow:

1. Main parser creates base parser service
2. After loading packages, calls ParseGeneralInfo
3. Base parser populates swagger.Info and security definitions
4. Main parser continues with route and type parsing

For more details, see [ARCHITECTURE.md](../../ARCHITECTURE.md).
