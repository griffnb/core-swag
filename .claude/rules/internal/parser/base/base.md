---
paths:
  - "internal/parser/base/**/*.go"
---

# Base Parser Service

## Overview

The Base Parser Service handles parsing of general API information from swagger annotation comments. It extracts API-level metadata including version, title, description, host, basePath, security definitions, tags, and external documentation from the main API file.

## Key Structs/Methods

### Core Types

- [Service](../../../../internal/parser/base/service.go#L40) - Main base parser service
- [Debugger](../../../../internal/parser/base/service.go#L35) - Debug logging interface

### Service Creation

- [NewService(swagger)](../../../../internal/parser/base/service.go#L47) - Creates new base parser with swagger spec

### Configuration

- [SetMarkdownFileDir(dir)](../../../../internal/parser/base/service.go#L54) - Sets directory for markdown documentation files
- [SetDebugger(debug)](../../../../internal/parser/base/service.go#L59) - Sets debug logger

### Main Parsing Methods

- [ParseGeneralAPIInfo(mainAPIFile)](../../../../internal/parser/base/service.go#L195) - Parses general API info from main file
- [ParseGeneralInfo(comments)](../../../../internal/parser/base/service.go#L64) - Parses API info from comment lines

### Internal Parsing Methods

- [parseSecurityDefinition(attr, comments, line)](../../../../internal/parser/base/security.go) - Parses security scheme definitions
- [parseSecurity(value)](../../../../internal/parser/base/security.go) - Parses security requirements
- [parseExtension(attr, value, tag)](../../../../internal/parser/base/extensions.go) - Parses custom extensions (x-*)
- [parseTagExtension(attr, value, tag)](../../../../internal/parser/base/extensions.go) - Parses tag-level extensions
- [setSwaggerInfo(attr, value)](../../../../internal/parser/base/info.go) - Sets swagger info properties
- [getMarkdownForTag(tagName)](../../../../internal/parser/base/markdown.go) - Loads markdown documentation

### Helper Functions

- [FieldsByAnySpace(s, limit)](../../../../internal/parser/base/helpers.go) - Splits string by whitespace
- [AppendDescription(desc, value)](../../../../internal/parser/base/helpers.go) - Appends to description with newline
- [parseMimeTypeList(value, target)](../../../../internal/parser/base/mime.go) - Parses MIME type list
- [isGeneralAPIComment(comments)](../../../../internal/parser/base/helpers.go) - Checks if comments are API-level

## Related Packages

### Depends On
- `go/ast` - AST representation
- `go/parser` - Go source parsing
- `go/token` - Token information
- `github.com/go-openapi/spec` - OpenAPI spec types

### Used By
- [parser.go](../../../../parser.go) - Main parser uses base service for API metadata
- [gen/gen.go](../../../../gen/gen.go) - Code generator uses parsed API info

## Docs

No dedicated README exists. Documentation is in godoc comments.

## Related Skills

No specific skills are directly related to this internal package.

## Usage Example

```go
// Create swagger spec
swagger := &spec.Swagger{
    SwaggerProps: spec.SwaggerProps{
        SecurityDefinitions: make(map[string]*spec.SecurityScheme),
    },
}

// Create base parser
baseParser := base.NewService(swagger)
baseParser.SetMarkdownFileDir("./docs")

// Parse main API file
err := baseParser.ParseGeneralAPIInfo("./main.go")
if err != nil {
    return err
}

// Swagger spec now contains:
// - swagger.Info (title, version, description, contact, license)
// - swagger.Host
// - swagger.BasePath
// - swagger.Schemes
// - swagger.SecurityDefinitions
// - swagger.Security
// - swagger.Tags
// - swagger.Consumes
// - swagger.Produces
```

## Supported Annotations

### API Metadata
- `@title` - API title
- `@version` - API version
- `@description` - API description (multi-line supported)
- `@description.markdown` - Load description from markdown file
- `@termsofservice` - Terms of service URL
- `@contact.name`, `@contact.url`, `@contact.email` - Contact information
- `@license.name`, `@license.url` - License information

### Server Information
- `@host` - API host
- `@basepath` - Base path prefix
- `@schemes` - Supported schemes (http, https, ws, wss)

### Content Types
- `@accept` - Accepted request content types
- `@produce` - Produced response content types

### Tags
- `@tag.name` - Define tag
- `@tag.description` - Tag description
- `@tag.description.markdown` - Load tag description from markdown
- `@tag.docs.url` - External documentation URL
- `@tag.docs.description` - External docs description

### Security
- `@securitydefinitions.basic` - HTTP Basic auth
- `@securitydefinitions.apikey` - API Key auth
- `@securitydefinitions.oauth2.application` - OAuth2 application flow
- `@securitydefinitions.oauth2.implicit` - OAuth2 implicit flow
- `@securitydefinitions.oauth2.password` - OAuth2 password flow
- `@securitydefinitions.oauth2.accesscode` - OAuth2 authorization code flow
- `@security` - Global security requirements

### Extensions
- `@x-*` - Custom extensions
- `@tag.x-*` - Tag-level extensions

### External Documentation
- `@externaldocs.description` - External docs description
- `@externaldocs.url` - External docs URL

## Design Principles

1. **Comment-Driven**: All API metadata comes from structured comments
2. **Markdown Support**: Can load descriptions from external markdown files
3. **Multi-Line Support**: Descriptions can span multiple comment lines
4. **MIME Type Aliases**: Supports shortcuts like "json" → "application/json"
5. **Security Definitions**: Full OAuth2 and API key support
6. **Extension Support**: Handles custom x-* extensions at all levels
7. **Error Recovery**: Continues parsing on individual annotation errors

## Common Patterns

- Always parse the main API file first before parsing routes
- Use markdown files for lengthy descriptions instead of comments
- Set debug logger during development to trace parsing issues
- Security definitions must be defined before they're referenced in @security
- Tags can be defined inline or via @tag annotations
