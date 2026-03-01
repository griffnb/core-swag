# Naming/Reference Resolution Bugs - Requirements

## Introduction

The core-swag OpenAPI documentation generator has three interrelated bugs in how it resolves type names to `$ref` values and definition keys. The system sometimes uses short (unqualified) names like `Properties` when it should use fully qualified names like `phone.Properties`. This causes ambiguous redirect warnings when multiple packages define types with the same name, unknown ref errors when curly-brace annotation patterns fall through to raw ref creation, and incorrect `$ref` targets when same-package types in `StructField[T]` generics are not package-qualified.

These bugs manifest in real-world projects as:
1. `Ambiguous redirect for Properties: phone.Properties vs account_identity_insurance.Properties`
2. `Skipping unknown ref utilities.object{new_id=string}`
3. Incorrect `$ref: "#/definitions/Properties"` instead of `$ref: "#/definitions/phone.Properties"`

---

## Requirement 1: Qualify same-package type parameters in processStructField

**User Story:** As a developer using core-swag, I want `StructField[T]` type parameters from the same package to be fully qualified with their package name, so that `$ref` values in the generated OpenAPI spec are always unambiguous.

### Acceptance Criteria

1.1. **When** `processStructField` encounters a `StructField[T]` where `T` is a type from the same package (no `/` or `.` in the extracted type name), **the system shall** prefix the type name with the base package name (e.g., `Properties` becomes `phone.Properties`).

1.2. **When** `processStructField` sets `f.TypeString` for a same-package type parameter, **the system shall** use the format `{arrayPrefix}{basePackageName}.{typeName}` where `basePackageName` comes from `c.basePackage.Name()`.

1.3. **When** `c.basePackage` is nil (no package context available), **the system shall** fall back to using the unqualified type name as it does today (preserving backward compatibility).

1.4. **When** `processStructField` encounters a `StructField[T]` where `T` already contains a package qualifier (has `.` or `/`), **the system shall not** add an additional package prefix (existing behavior must be preserved).

1.5. **When** the same-package type is wrapped in array (`[]`) or pointer (`*`) prefixes, **the system shall** preserve those prefixes while still qualifying the type name (e.g., `[]Properties` becomes `[]phone.Properties`).

1.6. **When** this fix is applied, **the system shall** continue to extract subfields from the type by calling `ExtractFieldsRecursive` with the correct package and type name (existing recursive extraction must not break).

---

## Requirement 2: Remove ambiguous unqualified redirects

**User Story:** As a developer using core-swag, I want the system to avoid creating ambiguous redirect definitions, so that types with the same short name in different packages don't silently resolve to the wrong definition.

### Acceptance Criteria

2.1. **When** `addUnqualifiedRedirects` detects that multiple qualified definitions map to the same unqualified name (e.g., `phone.Properties` and `account_identity_insurance.Properties` both map to `Properties`), **the system shall** skip creating a redirect for that unqualified name entirely (neither definition should win).

2.2. **When** an ambiguous redirect is detected, **the system shall** log a debug message listing all conflicting qualified names for the unqualified name.

2.3. **When** only one qualified definition maps to a given unqualified name, **the system shall** create the redirect as it does today (existing non-ambiguous behavior preserved).

2.4. **When** the unqualified name already exists as its own definition in `swagger.Definitions`, **the system shall not** create a redirect (existing behavior preserved).

2.5. **When** Bug 1 (Requirement 1) is fixed, the number of ambiguous redirect situations **should** decrease because more types will already have qualified names in their `$ref` values, reducing reliance on unqualified redirects.

---

## Requirement 3: Prevent curly braces from entering ref names

**User Story:** As a developer using core-swag, I want the system to properly handle annotation patterns like `utilities.object{new_id=string}` without creating malformed `$ref` values, so that the generated OpenAPI spec is valid.

### Acceptance Criteria

3.1. **When** `buildSchemaWithPackageAndPublic` detects a `{` in the `dataType` and routes to `buildAllOfResponseSchema`, and `ParseCombinedType` successfully parses the combined type syntax, **the system shall** produce a valid AllOf schema (existing behavior preserved).

3.2. **When** `ParseCombinedType` fails to parse the combined type (returns an error), **the system shall not** fall back to `buildSchemaForTypeWithPublic` with the raw curly-brace-containing string. Instead, it **shall** log a warning and return nil or an empty object schema.

3.3. **When** `buildSchemaForTypeWithPublic` receives a dataType containing curly braces `{` or `}`, **the system shall** reject it and return a generic object schema instead of creating a `$ref` with URL-encoded curly braces.

3.4. **When** `buildSchemaForType` (in `struct_field.go`) encounters a type string containing curly braces, **the system shall** treat it as a malformed type and return a generic object schema (the existing unbalanced bracket check should be extended to cover curly braces).

3.5. **When** a curly-brace pattern is rejected, **the system shall** log a debug/warning message indicating the malformed type name was skipped.

---

## Requirement 4: Maintain backward compatibility and test integrity

**User Story:** As a developer maintaining core-swag, I want all existing tests to continue passing after these fixes, so that the changes don't introduce regressions.

### Acceptance Criteria

4.1. **When** all three fixes are applied, **the system shall** produce identical or improved output from `make test-project-1` and `make test-project-2` (no new errors, fewer warnings).

4.2. **When** all three fixes are applied, **the system shall** pass `TestRealProjectIntegration` in `testing/core_models_integration_test.go`.

4.3. **When** all three fixes are applied, **the system shall** pass all existing unit tests (`go test ./...`).

4.4. **The system shall** include new unit tests for each fix that validate the specific bug scenario and its resolution.

4.5. **When** types are already qualified correctly (existing code paths that use `getQualifiedTypeName`), **the system shall not** double-qualify them (no `phone.phone.Properties`).

---

## Requirement 5: Follow existing code patterns

**User Story:** As a developer maintaining core-swag, I want fixes to use existing helper functions and patterns, so that the codebase remains consistent and maintainable.

### Acceptance Criteria

5.1. **When** qualifying type names, **the system shall** reuse the existing `getQualifiedTypeName` function or follow its pattern (using `pkg.Name()` for the package prefix).

5.2. **When** adding validation, **the system shall** follow the existing pattern of bracket-depth checking in `buildSchemaForType` (lines 406-422 of `struct_field.go`).

5.3. **The system shall** keep changes minimal and focused -- no refactoring beyond what is necessary to fix the three bugs.

5.4. **The system shall** use `console.Logger.Debug` for all new log messages, consistent with existing logging patterns.

5.5. **The system shall** update go docs for any modified functions to reflect the new behavior.
