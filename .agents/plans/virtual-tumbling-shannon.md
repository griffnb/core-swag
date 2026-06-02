# Debug Off By Default

## Context

Today every `core-swag init` run emits verbose orchestrator/loader/registry debug output by default. The `--debug` CLI flag exists but only affects `console.Logger.DebugLevel`; the orchestrator path is plumbed through a separate `Debugger` interface (`gen.Config.Debugger` → `orchestrator.Config.Debug` → `loader.WithDebugger` / `registry.SetDebugger` / `base.SetDebugger`), and `cmd/core-swag/main.go` unconditionally constructs a `log.New(os.Stdout, ...)` and passes it down. Each service then guards every call site with `if cfg.Debug != nil`.

We want:
- **Default:** only true error states print (package/model not found, schema build failures, output type unsupported).
- **`--debug`:** the existing verbose traces return.
- **One logger.** Every service currently carries its own `Debugger` interface and plumbing — that's the actual bug. Collapse to a single package-level logger in `internal/console`.

## Approach

Centralize all logging on the existing `console.Logger` singleton. Delete the per-service `Debugger` interfaces, config fields, and setters. The CLI flips one bit (`console.Logger.DebugLevel`) and everything downstream becomes silent or verbose accordingly. Real errors go through a new `console.Logger.Error` method that always prints to stderr.

### 1. Extend `console.Logger`

`internal/console/debug.go`:
- Keep `DebugLevel int` and existing `Debug(format, args...)` (gated on `DebugLevel >= 1`, writes to stdout via `printf`).
- Add `Error(format string, args ...any)` — writes unconditionally to **stderr** via `fmt.Fprint(os.Stderr, Sprintf(format+"\n", args...))`. Template syntax (`$Red{...}`) still works.
- That's it. No `NoopDebugger`, no per-service interfaces — every caller uses `console.Logger.Debug(...)` / `console.Logger.Error(...)` directly.

### 2. Delete the `Debugger` plumbing from every service

Remove the interface, the config field, the setter, and the call-site guards in each of these:

- `internal/gen/gen.go`:
  - Remove `Debugger` interface (line 45-47), `Gen.debug` field (line 41), `Config.Debugger` field (line 71), and the `g.debug = config.Debugger` assignment in `Build`.
  - Replace `g.debug.Printf("Sanitizing ...")` (line 221) with `console.Logger.Debug("Sanitizing ...")`.
- `internal/orchestrator/service.go`:
  - Remove `Debugger` interface (line 55-57) and `Config.Debug` field (line 51).
  - Remove every `if s.config.Debug != nil` block and call `console.Logger.Debug(...)` directly.
- `internal/orchestrator/schema_builder.go`: same — drop guards, call `console.Logger.Debug(...)`.
- `internal/loader`: remove `WithDebugger` option, the field on the service, and guards. Replace usages with `console.Logger.Debug(...)`.
- `internal/registry`: remove `SetDebugger` method, the field, and guards. Same replacement.
- `internal/parser/base`: remove `SetDebugger` method, the field, and guards. Same replacement.

Verify completeness with:
```
grep -rn "Debugger\|SetDebugger\|WithDebugger\|Debug != nil\|Debugger != nil" internal/ cmd/
```
After the change this should only match `console.Logger.Debug(...)` calls and the new `Error` method.

### 3. CLI flips the single bit

`cmd/core-swag/main.go`:
- In `initAction`: when `ctx.Bool(debugFlag)` is true, set `console.Logger.DebugLevel = 1`. Otherwise leave it at 0.
- Delete the `logger := log.New(os.Stdout, ...)` block (lines 207-210) and the `Debugger: logger` field on the `gen.Config` literal.
- `--quiet` (`io.Discard`) currently only affected the deleted logger; since `console.Logger.Debug` already does nothing when `DebugLevel == 0`, `--quiet` becomes redundant. Either drop the flag or leave it as a no-op for back-compat — recommend leaving the flag declared but unused (one-line note) to avoid breaking scripted invocations.

### 4. Promote real error states to `console.Logger.Error`

These four messages represent actual failures, not progress, and should always print:

- `internal/orchestrator/schema_builder.go:163-165` — `BuildAllSchemas FAILED for %s (pkg=%s): %v`
- `internal/orchestrator/schema_builder.go:227-233` — `Skipping unknown ref %s (not in registry) referenced by %s` (model/package not found)
- `internal/orchestrator/schema_builder.go:250-252` — `BuildSchema FAILED for %s: %v` (non-struct build error)
- `internal/gen/gen.go:236` — already `log.Printf("output type '%s' not supported", ...)`; switch to `console.Logger.Error(...)` for consistency.

`preWarmPackages failed (non-fatal)` at `schema_builder.go:92-94` stays on `console.Logger.Debug` (explicitly non-fatal).

### 5. Silence converter `log.Printf` noise

`internal/parser/route/converter.go` lines 76, 79, 113, 125, 133, 267 — leftover investigative WARNING/INFINITY messages. `gen.sanitizeSwaggerSpec` already scrubs the bad values, so they add nothing user-facing. Convert each `log.Printf(...)` to `console.Logger.Debug(...)`. Drop the `"log"` import if no other use remains.

## Files Modified

- `internal/console/debug.go` — add `Error` method
- `cmd/core-swag/main.go` — gate `console.Logger.DebugLevel` on `--debug`; drop the constructed logger
- `internal/gen/gen.go` — remove `Debugger` interface/field/config; switch logging to `console.Logger`
- `internal/orchestrator/service.go` — remove `Debugger` interface and `Config.Debug`; route logging through `console.Logger`
- `internal/orchestrator/schema_builder.go` — same; promote failure messages to `console.Logger.Error`
- `internal/loader/*` — remove `WithDebugger` and field; use `console.Logger`
- `internal/registry/*` — remove `SetDebugger` and field; use `console.Logger`
- `internal/parser/base/*` — remove `SetDebugger` and field; use `console.Logger`
- `internal/parser/route/converter.go` — `log.Printf` → `console.Logger.Debug`
- `.claude/rules/internal/console/console.md` — document the new `Error` entry point and the central-logging rule

## Verification

1. `go build ./...` clean.
2. `grep -rn "Debugger\|SetDebugger\|WithDebugger" internal/ cmd/` returns nothing.
3. `make test-project-1` produces a clean stdout — no `Orchestrator: …`, no `Generate swagger docs....`, no `INFINITY DETECTED …`.
4. `./core-swag init --debug -g main.go -d ./testing/testdata/simple` brings back the full verbose trace.
5. Force a missing-ref scenario (e.g. point at a project with an unresolvable `@Success` ref); confirm `Skipping unknown ref …` appears on **stderr** without `--debug`.
6. `go test ./...` passes — particularly `testing/core_models_integration_test.go` (`TestRealProjectIntegration`, `TestCoreModelsIntegration`).
