# Fix `x-path` to use module-relative Go import path

## Context

Today the generated OpenAPI spec's `x-path` extension is an absolute filesystem path:

```
"x-path": "/Users/griffnb/projects/Crowdshield/atlas-go/atlas-go/internal/controllers/accounts/auth.go"
```

That value leaks the local machine layout and is not portable across machines or CI runs. The intent of `x-path` is to point at the source file in a stable, repo-relative way. It should be the Go module-relative path:

```
"x-path": "github.com/Crowdshield/atlas-go/internal/controllers/accounts/auth.go"
```

i.e. `<go module import path of the package>/<filename>`.

## Root Cause

`x-path` is written from `route.FilePath` in `internal/parser/route/converter.go:33-35`. `FilePath` is set by `Service.operationToRoutes` in `internal/parser/route/service.go:158` from `op.filePath`, which is whatever caller passed into `ParseRoutes(astFile, filePath, fset)`.

The single non-test caller is `internal/orchestrator/routes_parallel.go:46`, which passes `fileInfo.Path` — the absolute on-disk path from the loader.

The loader (`internal/loader`) already has the data we need on every `*AstFileInfo`:
- `Path` — absolute filesystem path (e.g. `/Users/.../accounts/auth.go`)
- `PackagePath` — Go import path of the containing package (e.g. `github.com/Crowdshield/atlas-go/internal/controllers/accounts`)

Both loader strategies populate `PackagePath`:
- `gopackages.go:102` — sets it directly from `pkg.PkgPath` (always the canonical Go import path).
- `loader.go:90` — sets it from `getPkgName(absDir)` walking under `getModuleInfo`'s detected module root.

So the module-relative path is just `PackagePath + "/" + filepath.Base(Path)`. No new module-detection logic is needed.

## Change

Construct the module-relative display path at the orchestrator call site and hand it to `ParseRoutes` as the `filePath` argument. `route.FilePath` becomes a "display path" (module-relative) rather than an on-disk path. This is consistent with both existing consumers of the field:

- `internal/parser/route/converter.go:34` — writes the value verbatim to `x-path` (this is the field we want to change).
- `internal/orchestrator/refs.go:47-58` — only uses `filepath.Base`-style stripping for human-readable log messages. Module paths use `/` as the separator, so the basename extraction still produces `auth.go` correctly.

No `*os.Open` / `os.Stat` / disk reads happen against `route.FilePath`, so dropping the absolute path is safe.

### Files to modify

**`internal/orchestrator/routes_parallel.go`** (the only non-test caller of `ParseRoutes`)

At line 46, replace:

```go
routes, err := s.routeParser.ParseRoutes(astFile, fileInfo.Path, fileInfo.FileSet)
```

with construction of the module-relative path first, then pass it in. Fallback to the absolute path when `PackagePath` is unavailable (e.g. files outside any module, or when `getModuleInfo` failed in the filepath-walk loader). Use the same `displayPath` for the internal `fileRoutes.filePath` at line 56 so logs/sort keys stay consistent.

Sketch:

```go
displayPath := fileInfo.Path
if pkg := fileInfo.PackagePath; pkg != "" && pkg != "." {
    displayPath = pkg + "/" + filepath.Base(fileInfo.Path)
}
routes, err := s.routeParser.ParseRoutes(astFile, displayPath, fileInfo.FileSet)
```

Add `path/filepath` to the existing import block.

### Files NOT modified

- `internal/parser/route/service.go`, `domain/route.go`, `converter.go` — signature and field semantics stay the same; the value flowing through them just changes meaning from "absolute path" to "display path".
- `internal/loader/*` — no changes; we are consuming data it already exposes.
- `internal/parser/route/*_test.go` — these pass `"test.go"` as `filePath`. Behavior is unchanged for them; `x-path` will be `"test.go"` exactly as before.
- `internal/parser/route/converter_test.go` and `registration_test.go` — already use literal `"/src/handlers/user.go"`; still pass through unchanged.

## Verification

1. Unit tests:
   - `go test ./internal/parser/route/...`
   - `go test ./internal/orchestrator/...`
2. End-to-end on a real project (per `.claude/CLAUDE.md`):
   - `make test-project-1` → inspect `testing/test-project-1/swagger.json`; grep for `x-path` and confirm values look like `github.com/<owner>/<repo>/path/to/file.go`, not `/Users/...`.
   - `make test-project-2` → same check on `testing/test-project-2/swagger.json`.
3. Sanity check the integration suite:
   - `go test ./testing/... -run 'TestRealProjectIntegration|TestCoreModelsIntegration'`
4. Confirm fallback behavior is acceptable: any file where `PackagePath` is empty (rare — only when the loader couldn't determine module info) will still emit the absolute path as before, so no spec entries get a blank `x-path`.
