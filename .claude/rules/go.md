---
paths: *.go
---


# Go Development Instructions

Follow idiomatic Go practices and community standards when writing Go code. These instructions are based on [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), and [Google's Go Style Guide](https://google.github.io/styleguide/go/).

## General Instructions

- Write simple, clear, and idiomatic Go code
- Favor clarity and simplicity over cleverness
- Follow the principle of least surprise
- Keep the happy path left-aligned (minimize indentation)
- Return early to reduce nesting
- Make the zero value useful
- Document exported types, functions, methods, and packages
- Use Go modules for dependency management

## Naming Conventions

### Packages

- Use lowercase, single-word or snake_case package names
- Avoid hyphens, or mixedCaps
- Choose names that describe what the package provides, not what it contains
- Avoid generic names like `util`, `common`, or `base`
- Package names should be singular, not plural

### Variables and Functions

- Use mixedCaps or MixedCaps (camelCase) rather than underscores
- Keep names short but descriptive
- Use single-letter variables only for very short scopes (like loop indices)
- Exported names start with a capital letter
- Unexported names start with a lowercase letter
- Avoid stuttering (e.g., avoid `http.HTTPServer`, prefer `http.Server`)

### Interfaces

- Name interfaces with -er suffix when possible (e.g., `Reader`, `Writer`, `Formatter`)
- Single-method interfaces should be named after the method (e.g., `Read` → `Reader`)
- Keep interfaces small and focused

### Constants

- Use CAPS for exported constants
- Use mixedCaps for unexported constants
- Group related constants using `const` blocks
- Use typed constants for better type safety

## Code Style and Formatting

### Formatting

- Always use MCP #code_tools to format code
- Keep line length reasonable (no hard limit, but consider readability)
- Focus on readability over cleverness


### Error Handling

- Check errors immediately after the function call
- Don't ignore errors using `_` unless you have a good reason (document why)
- Wrap errors with context using `errors.Wrapf`
- Create custom error types when you need to check for specific errors
- Place error returns as the last return value
- Name error variables `err`
- Keep error messages lowercase and don't end with punctuation

## Architecture and Project Structure

### Package Organization

- Follow standard Go project layout conventions
- Group related functionality into packages
- Avoid circular dependencies
- use files and file naming to break packages into common parts


### Type Definitions
- Use structs over maps for type safety
- Use struct tags for JSON
- Prefer explicit type conversions using tools.ParseStringI(x), tools.ParseIntI(x) instead of .(sometype)

### Pointers vs Values

- Use pointers for large structs or when you need to modify the receiver
- Use values for small structs and when immutability is desired
- Be consistent within a type's method set
- Default to pointer receivers for structs in general

### Interfaces and Composition
- Accept interfaces, return concrete types
- Keep interfaces small (1-3 methods is ideal)
- Use embedding for composition
- Define interfaces close to where they're used, not where they're implemented
- Don't export interfaces unless necessary

## Concurrency

### Goroutines
- Comment where a go routine would be helpful. Alaways ask before implementing concurrency. When its agreed to, follow the best practices below
- Don't create goroutines in libraries; let the caller control concurrency
- Always know how a goroutine will exit
- Use `sync.WaitGroup` or `errgroup.WithContext` to wait for goroutines
- Avoid goroutine leaks by ensuring cleanup

## Error Handling Patterns

### Creating Errors

- Use `errors.Errorf` for dynamic errors
- Create custom error types for domain-specific errors
- Export error variables for sentinel errors
- Use `errutil.Find` and `errutil.As` for error checking error type

### Error Propagation
- Add context only if its useful before returning using `errors.Wrapf()`
- Always return errors up to the controller unless it needs to be handled locally, leave comment on why
- Only log errors when its no longer bubbled, use `log.ErrorContext(err,ctx)`

## API Design

### HTTP Endpoints

- Follow existing code generated patterns in the controllers

### JSON APIs

- Use struct tags to control JSON marshaling
- For public endpoints, struct tags need `public:"view|edit"` to control what is sent/received
- Use `omitempty` for optional fields
- Validate input data
- Use pointers for optional fields
- Consider using `json.RawMessage` for delayed parsing
- Handle JSON errors appropriately

## Security Best Practices

### Input Validation
- Validate all external input
- Use strong typing to prevent invalid states
- Sanitize data before using in SQL queries, avoid custom queries and use the model loaders if possible

## Documentation

### Code Documentation

- Document all exported symbols
- Start documentation with the symbol name
- Keep documentation close to code
- Update documentation when code changes
- Dont document the obvious, dont put examples in documents
- Make sure they are to the point

## Tools and Development Workflow

### Essential Tools
- Use ONLY MCP #code_tools tools for formatting / linting / testing.  It properly configures things

### Development Practices

- Run tests before committing
- Keep commits focused and atomic
- Write meaningful commit messages
- Review diffs before committing

## Common Pitfalls to Avoid

- Not checking errors
- Ignoring race conditions
- Creating goroutine leaks
- Not using defer for cleanup
- Modifying maps concurrently
- Not understanding nil interfaces vs nil pointers
- Forgetting to close resources (files, connections)
- Using global variables unnecessarily
- Over-using empty interfaces (`interface{}`)
- Not considering the zero value of types
- using `interface{}` intead of `any`
