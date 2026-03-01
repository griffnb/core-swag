---
paths:
  - "internal/parser/base/**/*.go"
---

# Base Parser Service

## Overview

The Base Parser Service handles parsing of general API information from swagger annotation comments. It extracts API-level metadata including version, title, description, host, basePath, security definitions, tags, and external documentation from the main API file.

## Key Structs/Methods

### Core Types

- [Debugger](../../../../internal/parser/base/service.go#L35) - Debug logging interface
- [Service](../../../../internal/parser/base/service.go#L40) - Main base parser service

### Service Creation & Configuration

- [NewService(swagger)](../../../../internal/parser/base/service.go#L47) - Creates new base parser with swagger spec
- [Service.SetMarkdownFileDir(dir)](../../../../internal/parser/base/service.go#L54) - Sets directory for markdown documentation files
- [Service.SetDebugger(debug)](../../../../internal/parser/base/service.go#L59) - Sets debug logger

### Main Parsing Methods

- [Service.ParseGeneralInfo(comments)](../../../../internal/parser/base/service.go#L64) - Parses API info from comment lines
- [Service.ParseGeneralAPIInfo(mainAPIFile)](../../../../internal/parser/base/service.go#L195) - Parses general API info from main file

### Helper Functions

- [FieldsFunc(s, f, n)](../../../../internal/parser/base/utils.go#L9) - Splits string by function with limit
- [FieldsByAnySpace(s, n)](../../../../internal/parser/base/utils.go#L56) - Splits string by whitespace with limit
- [AppendDescription(current, addition)](../../../../internal/parser/base/utils.go#L62) - Appends to description with newline

### Internal Methods

- [Service.setSwaggerInfo(attribute, value)](../../../../internal/parser/base/info.go#L13) - Sets swagger info properties
- [Service.getMarkdownForTag(tagName)](../../../../internal/parser/base/info.go#L39) - Loads markdown documentation
- [Service.parseSecurityDefinition(context, lines, index)](../../../../internal/parser/base/security.go#L10) - Parses security scheme definitions
- [Service.parseExtension(attribute, value, tag)](../../../../internal/parser/base/extensions.go#L12) - Parses custom extensions (x-*)
- [Service.parseTagExtension(attribute, value, tag)](../../../../internal/parser/base/extensions.go#L51) - Parses tag-level extensions
- [isGeneralAPIComment(comments)](../../../../internal/parser/base/helpers.go#L9) - Checks if comments are API-level
- [parseMimeTypeList(commentLine, mimeTypes)](../../../../internal/parser/base/helpers.go#L25) - Parses MIME type list
- [parseSecurity(commentLine)](../../../../internal/parser/base/helpers.go#L46) - Parses security requirements

## Related Packages

### Depends On
- `go/parser`, `go/token` - Go source parsing
- `github.com/go-openapi/spec` - OpenAPI spec types

### Used By
- [internal/orchestrator/service.go](../../../../internal/orchestrator/service.go) - Creates base parser and calls `ParseGeneralAPIInfo`

## Docs

- [Base Parser README](../../../../internal/parser/base/README.md)

## Related Skills

No specific skills are directly related to this package.
