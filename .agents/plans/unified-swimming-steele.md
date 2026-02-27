# Plan: Fix All Failing Tests

## Context

Running `go test ./...` shows failing packages. Goal: make all tests pass (excluding `internal/legacy_files` per user direction).

## Scope

| Package | Root Cause | Action |
|---|---|---|
| `internal/gen` | Wrong relative testdata paths | Fix paths |
| `internal/loader` | Wrong relative testdata paths | Fix paths |
| `internal/orchestrator` | Wrong relative testdata paths | Fix paths |
| `internal/legacy_files` | Wrong paths | **Skip (per user)** |
| `internal/parser/route` | Bug in `convertSpecSchemaToDomain` | Fix AllOf logic |
| `testing/` | Missing go.sum entries for transitive deps (pgx, dynamodb, etc.) | Run `go mod tidy` |

---

## Fix 1: Testdata Path Corrections

Testdata lives at `testing/testdata/` but tests use incorrect relative paths.

**`internal/gen/gen_test.go`** — Change `../testdata/` → `../../testing/testdata/` (all occurrences)

**`internal/loader/service_test.go`** — Change `../../testdata/` → `../../testing/testdata/` (all occurrences)

**`internal/orchestrator/debug_test.go`** — Change `../../testdata/` → `../../testing/testdata/` (all occurrences)

---

## Fix 2: AllOf Composition Bug in Route Parser

**File:** `internal/parser/route/allof.go` (lines 126-135)

**Bug:** `convertSpecSchemaToDomain` returns early after building AllOf entries, never extracting Properties from the override schema to top-level `domainSchema.Properties`.

**Fix:** After building AllOf, extract Properties from AllOf elements that have them (the second element is the override object with properties like `data`, `meta`, `count`).

---

## Fix 3: Testing Module Dependencies

**Directory:** `testing/`

Run `make tidy` (which sets private keys properly) to resolve missing go.sum entries for transitive deps of `core/lib` (pgx, dynamodb, mysql, strcase).

---

## Verification

1. `go test ./internal/gen/... ./internal/loader/... ./internal/orchestrator/...` — testdata tests pass
2. `go test -run TestAllOfComposition ./internal/parser/route/...` — AllOf tests pass
3. `cd testing && go test -run TestBuildAllSchemas -v` — schema tests pass
4. `go test ./...` — all main module packages pass
