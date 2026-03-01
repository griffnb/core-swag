# Plan: Remove `internal/legacy_files` Package

## Context

The `internal/legacy_files` folder contains ~26 files of ported legacy swaggo code. The new architecture has fully replaced this with the orchestrator + schema + route parser pipeline. Only 4 files still import `legacy_files` for a handful of small symbols. The docs.go generation (which was the main consumer of the runtime types Spec/Register) is no longer needed since it's not part of any OpenAPI spec output.

## Steps

### Step 1: Remove docs.go generation from `internal/gen/gen.go`

The `writeDocSwagger`, `writeGoDoc`, and `packageTemplate` are all about generating docs.go files that imported from `github.com/swaggo/swag`. This output type is no longer needed.

- **Remove** `writeDocSwagger` method (lines 270-309)
- **Remove** `writeGoDoc` method (lines 448-546)
- **Remove** `packageTemplate` var (lines 548-573)
- **Remove** `"go": gen.writeDocSwagger` from outputTypeMap (line 64)
- **Remove** docs.go-only Config fields: `GeneratedTime` (line 127), `LeftTemplateDelim` (line 142-143), `RightTemplateDelim` (line 145-146), `PackageName` (line 147-148)
- **Remove** unused imports that were only needed for docs.go generation: `"go/format"`, `"time"`, `"text/template"`, `"golang.org/x/text/cases"`, `"golang.org/x/text/language"` — verify each is unused after removal
- **Remove** `formatSource` method if only used by writeGoDoc

### Step 2: Move remaining needed symbols into appropriate packages

**In `internal/gen/gen.go`:**
- **Add** `const Version = "v1.16.7"`
- **Add** `const DefaultInstanceName = "swagger"`
- **Change** `Config.Debugger` type from `swag.Debugger` → `Debugger` (local interface at line 48)
- **Replace** all `swag.Name` references → `DefaultInstanceName` (lines 172, 277→deleted, 320, 355)
- **Remove** the `swag` import (line 21)

**In `internal/parser/field/helpers.go`:**
- **Add** `TransToValidCollectionFormat` function (8 lines, validates collection format strings)

### Step 3: Update `cmd/core-swag/main.go`

- **Replace** import `swag "github.com/griffnb/core-swag/internal/legacy_files"` with `"github.com/griffnb/core-swag/internal/parser/field"`
- **Replace** `swag.CamelCase` → `field.CamelCase` (lines 77, 78, 212)
- **Replace** `swag.SnakeCase` → `field.SnakeCase` (lines 78, 212)
- **Replace** `swag.PascalCase` → `field.PascalCase` (lines 78, 212)
- **Replace** `swag.TransToValidCollectionFormat` → `field.TransToValidCollectionFormat` (line 248)
- **Replace** `swag.Version` → `gen.Version` (line 299)
- **Update** outputTypes default value from `"go,json,yaml"` → `"json,yaml"` (line 89)
- **Update** usage text to remove docs.go reference (line 90)

### Step 4: Delete testdata that depends on legacy_files

These testdata dirs are used by tests that are already `t.Skip`'d as legacy:
- **Delete** `testing/testdata/delims/` (entire dir)
- **Delete** `testing/testdata/quotes/` (entire dir)

### Step 5: Clean up `internal/gen/gen_test.go`

- **Remove** `"go"` from the `outputTypes` var (line 24): change to `[]string{"json", "yaml"}`
- **Remove** skipped legacy tests: `TestGen_BuildDescriptionWithQuotes` (lines 205-262), `TestGen_BuildDocCustomDelims` (lines 264-327), `TestGen_GeneratedDoc` (lines 622-654)
- **Update** any remaining test that checks for docs.go output to not expect it
- **Remove** now-unused test imports (`"os/exec"`, `"plugin"` if only used by deleted tests)

### Step 6: Delete `internal/legacy_files/` folder

With all references removed, delete the entire directory (~26 source files + test files).

### Step 7: Clean up

- `go mod tidy` to remove any orphaned dependencies
- Remove `testdata/delims` and `testdata/quotes` entries from `.gitignore` (lines 3-6)

## Files Modified

| File | Action |
|------|--------|
| `internal/gen/gen.go` | Remove docs.go generation; add Version + DefaultInstanceName; fix Debugger type; remove swag import |
| `internal/gen/gen_test.go` | Remove "go" from outputTypes; delete 3 skipped legacy tests; clean up |
| `internal/parser/field/helpers.go` | Add `TransToValidCollectionFormat` |
| `cmd/core-swag/main.go` | Switch imports from legacy_files to field + gen |
| `testing/testdata/delims/` | Delete entire directory |
| `testing/testdata/quotes/` | Delete entire directory |
| `internal/legacy_files/` | Delete entire directory |
| `.gitignore` | Remove testdata/quotes/docs and testdata/delims entries |

## Verification

1. `go build ./...` — confirms no compilation errors
2. `go vet ./...` — confirms no issues
3. `go test ./internal/gen/...` — gen tests pass
4. `go test ./cmd/...` — CLI tests pass (if any)
5. `make test-project-1` and `make test-project-2` — real project integration still works
6. Verify `internal/legacy_files/` no longer exists
7. Verify no references to `legacy_files` or `swaggo/swag` remain in Go source files
