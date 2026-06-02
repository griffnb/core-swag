# Fix missing-ref bug in test-project-3 (`object_sync.TaskUpdate` / `airtable_sync.TaskUpdate`)

## Context

`make test-project-3` produces a swagger.json with two `$ref` lookups that have no matching definition:
- `airtable_sync.TaskUpdate`
- `object_sync.TaskUpdate`

`make test-project-1` (atlas-go) does not have this problem, even though both projects reference these types in `@Success` annotations on routes in `internal/controllers/sync/`. The difference looked superficial, but the root cause is in core-swag's dependency loader, not in the user projects.

### Root cause (verified)

The CLI invocation uses `--parseInternal -pd` and defaults `parseGoList=true`, so the loader takes the path:
`Service.Parse → loader.LoadDependencies → loadDependenciesWithGoList → listPackages → go list -json -e -deps` (see `internal/loader/dependency.go:33` and `internal/loader/golist.go:48`).

`listOnePackages` runs `go list -deps` with `cmd.Dir = <searchDir>` and **no package argument**. That makes `go list` resolve only the package directly in `<searchDir>`, plus its transitive deps. It does NOT recurse into sub-packages.

For `./internal/controllers`, both projects contain a `router.go` at that path that imports many sub-controllers:
- atlas-go's `controllers/router.go` imports `controllers/sync` (line 100)
- go-litigate's `controllers/router.go` does **not** import `controllers/sync` — sync is wired up elsewhere

Result for test-project-3:
- `LoadSearchDirs` walks `./internal/controllers/...` recursively and parses `controllers/sync/admin.go`, so the `@Success ... {object} object_sync.TaskUpdate` annotation is parsed and the `$ref` is emitted with a fully-resolved `TypePath`.
- But `LoadDependencies` (go-list path) never sees `controllers/sync` because the cwd-root package doesn't import it. So `internal/services/object_sync` and `internal/services/airtable_sync` are never parsed, and the registry never gets `TaskUpdate` types.
- `buildDemandDrivenSchemas → resolveRef` then logs `Orchestrator: Skipping unknown ref ... (not in registry)` and the definition is dropped.

Confirmed empirically:
```
cd .../go-litigate/backend
go list -deps -e ./internal/controllers           # no object_sync, no airtable_sync
go list -deps -e ./internal/controllers/...       # both present
```

## Fix

Change `loadDependenciesWithGoList` to invoke `go list` with the recursive `./...` pattern so every sub-package under each search dir contributes to dependency resolution — independent of whether the directory-root package imports those sub-packages.

### File to modify

- `internal/loader/dependency.go:33`
  Change:
  ```go
  pkgs, err := listPackages(context.Background(), dirs, nil, "-deps")
  ```
  to:
  ```go
  pkgs, err := listPackages(context.Background(), dirs, nil, "-deps", "./...")
  ```
  `listPackages` already passes its variadic `args` through to `listOnePackages`, which appends them to `go list -json -e`. The resulting command becomes `go list -json -e -deps ./...` with `cwd=<searchDir>`, which lists every package under that dir plus all transitive deps. Deduplication happens in `listPackages` via `pkgMap[pkg.Dir]`, so other callers and the rest of the pipeline are unaffected.

No code changes are needed in:
- `LoadWithGoPackages` (already uses `absDir+"/..."` at `internal/loader/gopackages.go:28`)
- The depth-based fallback (`loadDependenciesWithDepth`) — used only when `useGoList=false`, which is not the CLI default; out of scope for this fix.
- The orchestrator or schema builder. Once the registry contains the type, the existing `FindTypeSpecByFullPath` lookup at `internal/orchestrator/schema_builder.go:212` resolves the ref correctly (the route parser already sets `RefInfo.TypePath` to the full import path via the controller's imports).

## Verification

1. Rebuild and re-run both projects, confirm the missing-ref count goes from 2 → 0 for project-3 and stays at 0 for project-1:
   ```bash
   make test-project-1
   make test-project-3
   python3 -c '
   import json, re
   for p in ["test-project-1", "test-project-3"]:
       s = json.load(open(f"testing/{p}/swagger.json"))
       defs = set(s.get("definitions", {}))
       refs = set(re.findall(r"\"\$ref\":\s*\"#/definitions/([^\"]+)\"", json.dumps(s)))
       missing = sorted(refs - defs)
       print(p, len(missing), missing)
   '
   ```
   Expect: `test-project-1 0 []`, `test-project-3 0 []`.

2. Confirm the `object_sync.TaskUpdate` and `airtable_sync.TaskUpdate` definitions are now present in `testing/test-project-3/swagger.json` with the expected fields (`model`, `total`, `current`, `current_model_number`, `total_models`, `error`).

3. Run the existing integration test guard to ensure project-1 hasn't regressed:
   ```bash
   go test ./testing -run TestRealProjectIntegration -v
   ```

4. Run the broader unit suite to ensure no unintended dependency-loading regression:
   ```bash
   make test
   ```

5. Add a regression unit test in `internal/loader/dependency_test.go` (new or existing file in that package) that asserts `loadDependenciesWithGoList` invokes `go list` with `./...`. A black-box test using a small fixture module where a sub-package imports an otherwise-unreferenced service package will fail before the fix and pass after.

## Change log entry

Append to `.agents/change_log.md` describing: what was tried (running with default `-pd`), why it didn't surface deps under sub-package directories that aren't reached from the dir-root package, and the fix (adding `./...`).
