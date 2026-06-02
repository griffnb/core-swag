# Plan: Fix non-deterministic / wrong output for same-name, different-path package collisions

## Context

`swagger.json` changes between runs, and for some types the **wrong object** is emitted.
Earlier hypotheses (Go map-key render order, flaky package loading) were ruled out by
inspection — output ordering is already deterministic (`encoding/json` sorts map keys, go-openapi
sorts `Properties`, `required` follows field declaration order).

The real cause is a **type-name collision across two packages that share a Go package name** but
live at different import paths:
- `github.com/CrowdShield/atlas-go/internal/services/atlasmail`
- `github.com/CrowdShield/atlas-go/internal/integrations/atlasmail`

Both define `InboxThread` and `Thread`, so both are marked `NotUnique` in the registry. A
controller (`internal/controllers/inbox_threads/admin.go`) imports the **services** one and
annotates `// @Success 200 {object} response.SuccessResponse{data=[]atlasmail.InboxThread}`.

### Evidence in `testing/test-project-1/swagger.json`
- Route `$ref` is the **short** name `#/definitions/atlasmail.InboxThread`.
- `definitions` contains BOTH the short `atlasmail.InboxThread` AND the canonical full-path
  `github_com_CrowdShield_atlas-go_internal_services_atlasmail.InboxThread`, plus
  `...internal_integrations_atlasmail.Thread`, etc.

The short-name definition is built from whichever `NotUnique` candidate the registry's map-order
fallback returns — non-deterministic, often the wrong package. Correct behavior: the route `$ref`
must point to the **canonical full-path name of the package the file actually imports**
(services), and **no short-name definition should exist**.

Intended outcome: byte-identical `swagger.json` every run, with collisions resolved to the type
the referencing file imports.

## Root cause — two coordinated bugs

**Bug A — the `$ref` string is never canonicalized.**
`resolveTypePath` (`internal/parser/route/service.go:54-63`) already resolves
`atlasmail.InboxThread` to the correct **services** `TypeSpecDef` using the file's imports
(`registry.FindTypeSpec` → `findPackagePathFromImports`). But:
- `resolveOverrideTypeSchema` (`internal/parser/route/allof.go:121,166-177`) hardcodes the short
  ref `#/definitions/atlasmail.InboxThread` and never sets `TypePath`.
- `resolveTypePathsInSchema` (`internal/parser/route/response.go:288-307`) backfills `TypePath`
  but **never rewrites `Ref`**.
- `buildSchemaForTypeWithPublic` (`internal/parser/route/response.go:161-197`) emits the short
  `#/definitions/<qualifiedType>` ref (line 195) even though it has `typePath` in hand (line 184).
So the operation keeps the short ref; `converter.go SchemaToSpec` (`converter.go:212-213`) emits it
verbatim and discards `TypePath`.

**Bug B — the model builder emits a stray short-name definition.**
`BuildAllSchemas` (`internal/model/struct_field_lookup.go:687,719`) unconditionally stores the
schema under the short `packageName + "." + schemaName` key, then *additionally* registers the
canonical name (`:721-735`). For `NotUnique` types this leaves a short-name definition that should
not exist.

### Naming equivalence (verified — makes the fix safe)
For `NotUnique` types, `TypeSpecDef.TypeName()` (`internal/domain/types.go:61-68`) sanitizes
`PkgPath` (`/`,`.`,`\` → `_`) and appends the type name — producing the **identical** string as the
resolver's `makeFullPathDefName2` (`internal/orchestrator/name_resolver.go:83-98`). For unique
types both yield the short `pkg.Type`. Therefore a `$ref` built from `typeDef.TypeName()` (Bug A
fix) exactly matches the definition key produced by `ResolveDefinitionName(...)` (Bug B fix).

## Fix

### Fix A — canonicalize route `$ref`s from the resolved type (parser side)
1. **`internal/parser/route/service.go`** — have `resolveTypePath` also return the resolved
   `*domain.TypeSpecDef` (it already fetches it via `FindTypeSpec` at `:58`). Add a small helper,
   e.g. `resolveTypeRef(qualifiedType, file, isPublic) (ref, typePath string)` that returns the
   canonical ref name `typeDef.TypeName()` (+ `"Public"` when `isPublic`) and `typeDef.FullPath()`
   (+ `"Public"`). Keep `resolveTypePath` as a thin wrapper or update callers.
2. **`internal/parser/route/response.go`**
   - `buildSchemaForTypeWithPublic` (`:161-197`): build the ref from the resolved typeDef's
     `TypeName()` instead of the short `qualifiedType`. When resolution fails (typeDef nil), keep
     today's short-name fallback so unknown/external types are unaffected.
   - `resolveTypePathsInSchema` (`:288-307`): in the `Ref != "" && TypePath == ""` branch, when the
     typeDef resolves, set **both** `schema.TypePath` and rewrite `schema.Ref` to
     `#/definitions/<TypeName()>` (re-appending `Public`). This covers the AllOf-override inner refs
     produced by `allof.go`.
3. **`internal/parser/route/parameter.go` (`~:76-91`)**: body-param refs are built the same way and
   carry `TypePath`; apply the same canonicalization so body params with collisions resolve too.
4. **`internal/parser/route/allof.go`**: no direct change needed if the rewrite is centralized in
   `resolveTypePathsInSchema` (called at `allof.go:55`); verify the override items ref is rewritten.

Because `CollectReferencedTypes` (`internal/orchestrator/refs.go`) keys off `schema.Ref`, once refs
are canonical the demand-driven builder is asked for the canonical name and everything aligns.

### Fix B — store the definition only under its canonical name (model side)
**`internal/model/struct_field_lookup.go:687,719-735`**: compute `canonicalName` via the existing
`globalNameResolver.ResolveDefinitionName(pkgPath + "." + lookupType)` (+ `"Public"`), then store
`allSchemas[canonicalName] = schema` **once**. Only when there is no resolver / no `/` in `pkgPath`
(unique/local types) fall back to the short `fullSchemaName`. This removes the stray short-name
definition for `NotUnique` types while leaving unique types unchanged (their canonical name *is*
the short name).

### Why both are required
- Fix A alone: ref points to the correct canonical definition (already built at `:733`), but the
  orphan short-name definition lingers.
- Fix B alone: orphan removed, but the un-rewritten short ref dangles.
- Together: ref → canonical full-path name; exactly one correct, deterministic definition under it.

## Defensive determinism (secondary — keep small)
Also make the registry's short-name fallback deterministic so any *remaining* short-name lookup
(e.g. a type referenced without resolvable import context) can't flip between runs:
- `internal/registry/service.go:155-181` `FindTypeSpecByName`: collect all matching `NotUnique`
  candidates, prefer project-local, then pick the lexicographically-smallest `TypeName()` instead
  of returning the first map-iteration match.
- `internal/registry/types.go:235-243` `FindTypeSpec` renamed-type (`//@name`) fallback: same —
  collect and pick the smallest, don't return first.
This is a safety net; Fix A+B is the actual correctness fix.

## Critical files
- `internal/parser/route/service.go` — `resolveTypePath` returns typeDef / new `resolveTypeRef`
- `internal/parser/route/response.go` — `buildSchemaForTypeWithPublic`, `resolveTypePathsInSchema`
- `internal/parser/route/parameter.go` — body-param ref canonicalization
- `internal/model/struct_field_lookup.go` — store under canonical name only (`:687,719-735`)
- `internal/registry/service.go`, `internal/registry/types.go` — deterministic fallbacks (secondary)
- (reference) `internal/orchestrator/name_resolver.go`, `internal/domain/types.go`

## Tests (use the `testing` skill for Go unit tests)
- `internal/parser/route` test: a `domain.Schema` with `Ref="#/definitions/atlasmail.InboxThread"`
  and a file importing `.../services/atlasmail`, registry seeded with two `NotUnique` `InboxThread`
  defs (services + integrations) → assert the rewritten `Ref` is the services full-path canonical
  name, both base and `Public`.
- `internal/model` test: `BuildAllSchemas` for a `NotUnique` type with a `globalNameResolver` →
  assert the returned map contains ONLY the canonical key, not the short `pkg.Type` key; unique
  types still keyed short.
- `internal/registry` test: two `NotUnique` project-local defs with the same short name →
  `FindTypeSpecByName` is stable across calls (secondary fix).

## Verification (end-to-end)
1. `make test-project-1` 5× → `shasum testing/test-project-1/swagger.json` identical all 5.
2. `make test-project-2` 5× → identical shasum all 5 (confirm definition count no longer drifts).
3. Grep the output: no `"atlasmail.InboxThread"`/`"atlasmail.Thread"` short definitions remain; the
   route `$ref`s point to `..._services_atlasmail.InboxThread` (matching the controllers' import).
4. Integration tests `TestRealProjectIntegration` / `TestCoreModelsIntegration`
   (`testing/core_models_integration_test.go`), run with `-count=3`; the "all `$ref` references have
   definitions" subtest must stay at 0 missing (catches any newly-dangling refs from Fix B).
5. `go test ./internal/... -race`.
