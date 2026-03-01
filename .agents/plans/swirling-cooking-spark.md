# Plan: Remove `internal/legacy_files` Package

## Context

The `internal/legacy_files` folder contains ~26 files of ported legacy code (the old swaggo parser, operation handler, generics, packages, etc.). The new architecture has fully replaced this with the orchestrator + schema + route parser pipeline. However, 4 files still import `legacy_files` for a handful of small symbols. The goal is to relocate those few symbols and delete the entire folder.

## Active Dependency Surface (only 10 symbols across 4 files)

| Symbol | Used By | Already Exists Elsewhere? |
|--------|---------|--------------------------|
| `Name` (const `"swagger"`) | `gen/gen.go` (lines 172, 277, 320, 355) | No |
| `Debugger` (interface) | `gen/gen.go` Config.Debugger (line 75) | Yes - `gen/gen.go:48` has its own identical `Debugger` |
| `Spec` / `Register` | `gen/gen.go` packageTemplate string (lines 557, 571) | Only in template text referencing `github.com/swaggo/swag` — not a Go import |
| `CamelCase`, `PascalCase`, `SnakeCase` | `cmd/core-swag/main.go` (lines 77, 78, 212) | Yes - `internal/parser/field/types.go:22-29` |
| `TransToValidCollectionFormat` | `cmd/core-swag/main.go` (line 248) | No |
| `Version` (const) | `cmd/core-swag/main.go` (line 299) | No |
| `ReadDoc` | `testing/testdata/delims/main.go`, `testing/testdata/quotes/main.go` | No |

## Plan

### Step 1: Move `TransToValidCollectionFormat` into `internal/parser/field/helpers.go`

This is a simple validation function that fits naturally with the existing field helpers (which already has `TransToValidSchemeType`, `CheckSchemaType`, etc.).

- **Add** `TransToValidCollectionFormat` to `internal/parser/field/helpers.go`
- Small 8-line function, trivial move

### Step 2: Move `Version` constant into `internal/gen/gen.go`

The version is only used by the CLI to set `app.Version`. Place it directly in the gen package since gen is the public API surface for the CLI.

- **Add** `const Version = "v1.16.7"` to `internal/gen/gen.go`

### Step 3: Move `Name` constant into `internal/gen/gen.go`

Only used by gen.go itself for instance name defaulting and filename prefixing.

- **Add** `const DefaultInstanceName = "swagger"` to `internal/gen/gen.go` (better name than `Name`)
- **Update** 4 references from `swag.Name` to `DefaultInstanceName`

### Step 4: Fix `Config.Debugger` type in `gen/gen.go`

`gen.go:75` uses `swag.Debugger` but `gen.go:48` already defines an identical local `Debugger` interface. Just change the Config field to use the local one.

- **Change** `Debugger swag.Debugger` → `Debugger Debugger` on line 75

### Step 5: Remove the `swag` import from `gen/gen.go`

After steps 2-4, gen.go has no remaining references to legacy_files.

- **Remove** line 21: `swag "github.com/griffnb/core-swag/internal/legacy_files"`

### Step 6: Update `cmd/core-swag/main.go`

- **Change** import from `legacy_files` to `"github.com/griffnb/core-swag/internal/parser/field"`
- **Replace** `swag.CamelCase` → `field.CamelCase` (3 locations)
- **Replace** `swag.SnakeCase` → `field.SnakeCase` (2 locations)
- **Replace** `swag.PascalCase` → `field.PascalCase` (2 locations)
- **Replace** `swag.TransToValidCollectionFormat` → `field.TransToValidCollectionFormat`
- **Replace** `swag.Version` → `gen.Version` (already imports gen)

### Step 7: Update testdata files

The testdata files (`testing/testdata/delims/main.go` and `testing/testdata/quotes/main.go`) use `swag.ReadDoc()` which is part of the swagger runtime registry (`Swagger` interface, `Register`, `ReadDoc`, `Spec`). This runtime code is needed for generated `docs.go` files to work.

However, these are **test data** files that simulate what a real project looks like. The generated `docs.go` template already references `github.com/swaggo/swag` (external package), not our internal legacy_files. These testdata files should also point to `github.com/swaggo/swag` to match the generated output.

- **Change** import in both testdata files from `legacy_files` to `"github.com/swaggo/swag"`
- Verify `github.com/swaggo/swag` is in `go.mod` (it should be since the generated template references it)

### Step 8: Delete `internal/legacy_files/` folder

With all references removed, delete the entire directory.

### Step 9: Clean up `go.mod` / `go.sum`

- Run `go mod tidy` to remove any dependencies that were only needed by legacy_files

## Files Modified

| File | Action |
|------|--------|
| `internal/parser/field/helpers.go` | Add `TransToValidCollectionFormat` |
| `internal/gen/gen.go` | Add `Version`, `DefaultInstanceName` constants; fix Debugger type; remove swag import |
| `cmd/core-swag/main.go` | Switch import from legacy_files to field + gen |
| `testing/testdata/delims/main.go` | Switch import to `github.com/swaggo/swag` |
| `testing/testdata/quotes/main.go` | Switch import to `github.com/swaggo/swag` |
| `internal/legacy_files/` (entire dir) | Delete |

## Verification

1. `go build ./...` — confirms no compilation errors
2. `go vet ./...` — confirms no issues
3. `make test-project-1` and `make test-project-2` — confirms real project integration still works
4. `go test ./internal/gen/...` — confirms gen tests pass
5. `go test ./cmd/...` — confirms CLI tests pass
6. Verify `internal/legacy_files/` no longer exists
