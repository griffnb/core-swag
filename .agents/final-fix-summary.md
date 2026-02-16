# Complete Fix for Regular Struct Schema Generation

## Summary

Fixed the schema generation for regular Go structs. The issue was in `internal/parser/struct/field_processor.go`, which processes struct fields BEFORE the schema builder fallback path.

## The Real Problem

The orchestrator flow is:
1. **Struct Parser** (ParseFile → ParseStruct → processField) builds schemas and adds them to definitions
2. **Schema Builder** (BuildSchema) checks if already exists and returns early

So the schema builder fallback code I fixed earlier wasn't even being used! The struct parser was adding schemas first with wrong types.

## Root Causes in field_processor.go

### 1. Missing Extended Primitive Detection
`resolvePackageType` only checked for `time.Time`, `uuid.UUID`, `decimal.Decimal` but NOT `types.UUID` and other variants.

### 2. Losing Type Information
`resolveBasicType` converted unknown types to generic `"object"`, losing the actual type name needed for creating $refs.

### 3. Missing Package Qualifiers
Same-package types like `*Properties` weren't getting the `classification.` prefix added, so they couldn't be matched to definitions.

### 4. Generic Object Fallback
`resolvePackageType` returned `"object"` for any unrecognized package-qualified type, including enums and cross-package structs.

## Complete Solution

### File: `/internal/parser/struct/field_processor.go`

### Fix #1: Update `resolvePackageType` (lines 218-232)
**Before:**
```go
func resolvePackageType(fullType string) string {
    // Only checked a few specific types
    if fullType == "time.Time" { return "string" }
    if fullType == "uuid.UUID" { return "string" }
    // ...
    return "object"  // ❌ Lost type info
}
```

**After:**
```go
func resolvePackageType(fullType string) string {
    // Check for fields.* named types first
    if fieldsType := resolveFieldsType(fullType); fieldsType != "" {
        return fieldsType
    }

    // Check if it's an extended primitive
    if isExtendedPrimitive(fullType) {
        return fullType  // Preserve full name
    }

    // Return the full type name for ALL package-qualified types
    // This allows buildPropertySchema to create proper $refs
    return fullType  // ✅ Preserves "constants.ClassificationType", etc.
}
```

### Fix #2: Update `buildPropertySchema` (lines 327-353)
**Added:** Check for extended primitives BEFORE creating $refs:
```go
// Standard types
openAPIType := fieldType
if fieldType == "integer" || fieldType == "number" || fieldType == "string" || fieldType == "boolean" {
    schema.Type = []string{fieldType}
} else {
    // NEW: Check extended primitives FIRST
    if isPrimitiveTypeName(fieldType) || isExtendedPrimitive(fieldType) {
        baseType, format := getPrimitiveSchema(fieldType)
        schema.Type = []string{baseType}
        if format != "" {
            schema.Format = format
        }
    } else if strings.Contains(fieldType, ".") {
        // Create $ref for package-qualified types
        return *spec.RefSchema("#/definitions/" + fieldType)
    } else {
        schema.Type = []string{resolveBasicType(fieldType)}
    }
}
```

### Fix #3: Update `processField` (lines 95-102)
**Added:** Package qualifier for same-package struct types:
```go
// For same-package struct types (no dot, not primitive), add package qualifier
if file != nil && !strings.Contains(fieldType, ".") &&
   !isPrimitiveTypeName(fieldType) &&
   fieldType != "object" && fieldType != "array" && fieldType != "map" &&
   fieldType != "interface" && fieldType != "integer" && fieldType != "number" &&
   fieldType != "string" && fieldType != "boolean" {
    // Add package qualifier: "Properties" → "classification.Properties"
    fieldType = file.Name.Name + "." + fieldType
}
```

### Fix #4: Update `resolveBasicType` (lines 199-216)
**Changed:** Preserve type names instead of converting to "object":
```go
default:
    // Preserve the type name for struct types
    return typeName  // ✅ Returns "Properties" not "object"
}
```

### Fix #5: Add Helper Functions (lines 400-445)
**New functions:**
```go
// isExtendedPrimitive checks for time.Time, UUID, decimal types
func isExtendedPrimitive(typeName string) bool {
    cleanType := strings.TrimPrefix(typeName, "*")
    extendedPrimitives := map[string]bool{
        "time.Time": true,
        "types.UUID": true,
        "uuid.UUID": true,
        "github.com/google/uuid.UUID": true,
        "github.com/griffnb/core/lib/types.UUID": true,
        "decimal.Decimal": true,
        "github.com/shopspring/decimal.Decimal": true,
    }
    return extendedPrimitives[cleanType]
}

// getPrimitiveSchema returns OpenAPI type and format
func getPrimitiveSchema(typeName string) (schemaType, format string) {
    cleanType := strings.TrimPrefix(typeName, "*")
    switch cleanType {
    case "time.Time":
        return "string", "date-time"
    case "types.UUID", "uuid.UUID", ...:
        return "string", "uuid"
    case "decimal.Decimal", ...:
        return "number", ""
    default:
        return resolveBasicType(typeName), ""
    }
}
```

### Fix #6: Update Slice Element Handling (lines 298-341)
**Added:** Extended primitive checks for array elements:
```go
if isPrimitiveTypeName(elemType) || isExtendedPrimitive(elemType) {
    baseType, format := getPrimitiveSchema(elemType)
    elemSchema = &spec.Schema{
        SchemaProps: spec.SchemaProps{
            Type: []string{baseType},
        },
    }
    if format != "" {
        elemSchema.Format = format
    }
} else if strings.Contains(elemType, ".") {
    // Create $ref for package-qualified types
    elemSchema = spec.RefSchema("#/definitions/" + elemType)
}
```

## Complete Test Results

### Test Case: `classification.JoinedClassification`
```go
type JoinedClassification struct {
    ID           types.UUID                   `json:"id"`
    Name         string                       `json:"name"`
    Type         constants.ClassificationType `json:"type"`
    Properties   *Properties                  `json:"properties"`
    Priority     int                          `json:"priority"`
    PublicBlurbs *PublicBlurbs                `json:"public_blurbs"`
    IsRevokable  int                          `json:"is_revokable"`
}
```

### Generated Schema - PERFECT ✅
```json
{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "format": "uuid"
    },
    "name": {
      "type": "string"
    },
    "type": {
      "$ref": "#/definitions/constants.ClassificationType"
    },
    "properties": {
      "$ref": "#/definitions/classification.Properties"
    },
    "priority": {
      "type": "integer"
    },
    "public_blurbs": {
      "$ref": "#/definitions/classification.PublicBlurbs"
    },
    "is_revokable": {
      "type": "integer"
    }
  }
}
```

## All Issues Fixed

✅ **UUID types** (`types.UUID`) → `{"type": "string", "format": "uuid"}`
✅ **time.Time** → `{"type": "string", "format": "date-time"}`
✅ **decimal.Decimal** → `{"type": "number"}`
✅ **Enum types** (`constants.ClassificationType`) → `{"$ref": "#/definitions/constants.ClassificationType"}`
✅ **Same-package nested structs** (`*Properties`) → `{"$ref": "#/definitions/classification.Properties"}`
✅ **Cross-package nested structs** → Proper `$ref` with full path
✅ **Arrays of structs** → `{"type": "array", "items": {"$ref": "..."}}`
✅ **Arrays of primitives** → Proper element types with formats
✅ **Recursive nesting** → All nested definitions created
✅ **Pointer types** → Handled correctly (stripped for type checking)

## Why This Was Hard to Find

1. **Multiple Schema Building Paths**:
   - Schema Builder fallback (which I fixed first)
   - **Struct Parser** (the real culprit) ✓
   - Route parameter parsing

2. **Early Exit**: Schema Builder checks if schema already exists and returns early, so the fallback code wasn't running

3. **Debug Output**: My debug statements were in BuildSchema, which returned early before running the fallback

4. **Complex Flow**: orchestrator → struct parser → processField → buildPropertySchema had multiple type resolution layers

## Build & Test Commands

```bash
# Build
go install ./cmd/core-swag

# Test on actual project
cd /Users/griffnb/projects/Crowdshield/atlas-go
core-swag init -g "main.go" -d "./cmd/server,./internal/controllers,./internal/models" --parseInternal -pd -o "./swag_docs"

# Check result
cat swag_docs/swagger.json | jq '.definitions["classification.JoinedClassification"]'
```

## Files Modified

1. `/Users/griffnb/projects/core-swag/internal/parser/struct/field_processor.go`
   - Updated `resolvePackageType` (lines 218-232)
   - Updated `buildPropertySchema` (lines 262-368, especially 327-353)
   - Updated `processField` (lines 95-102)
   - Updated `resolveBasicType` (lines 199-216)
   - Updated slice handling (lines 298-341)
   - Added `isExtendedPrimitive()` (lines 400-417)
   - Added `getPrimitiveSchema()` (lines 419-433)

2. `/Users/griffnb/projects/core-swag/internal/schema/builder.go`
   - Fixed fallback path (for completeness, though not the main issue)
   - Added `buildFieldSchema()` method
   - Updated `getFieldType()` to return 3 values

## Related Documentation

- `.agents/change_log.md` - Updated with complete fix details
- `.agents/schema-builder-fix-summary.md` - Earlier attempt (fallback path)
- `.agents/primitive-types-update-summary.md` - Extended primitive support in domain package

## Conclusion

The issue was in the **struct parser's field processor**, not the schema builder. By updating the field processor to:
1. Properly detect extended primitives
2. Preserve type names instead of converting to "object"
3. Add package qualifiers for same-package types
4. Return full type names for creating $refs

All regular Go structs now generate correct OpenAPI schemas with proper UUIDs, nested struct references, and enum references.
