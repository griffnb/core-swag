package swag

import (
	"go/ast"
	"go/token"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/domain"
)

// Type aliases for backward compatibility.
// These allow external packages to continue using swag.TypeSpecDef, etc.
// while the implementation has been moved to internal/domain.

type (
	// TypeSpecDef is an alias for domain.TypeSpecDef.
	TypeSpecDef = domain.TypeSpecDef

	// AstFileInfo is an alias for domain.AstFileInfo.
	AstFileInfo = domain.AstFileInfo

	// PackageDefinitions is an alias for domain.PackageDefinitions.
	PackageDefinitions = domain.PackageDefinitions

	// ConstVariable is an alias for domain.ConstVariable.
	ConstVariable = domain.ConstVariable

	// EnumValue is an alias for domain.EnumValue.
	EnumValue = domain.EnumValue

	// Schema is an alias for domain.Schema.
	Schema = domain.Schema

	// ConstVariableGlobalEvaluator is an alias for domain.ConstVariableGlobalEvaluator.
	ConstVariableGlobalEvaluator = domain.ConstVariableGlobalEvaluator
)

// Constants re-exported for backward compatibility
const (
	// EnumVarNamesExtension x-enum-varnames
	enumVarNamesExtension = domain.EnumVarNamesExtension
	// EnumCommentsExtension x-enum-comments
	enumCommentsExtension = domain.EnumCommentsExtension
	// EnumDescriptionsExtension x-enum-descriptions
	enumDescriptionsExtension = domain.EnumDescriptionsExtension
)

// NewPackageDefinitions creates a new PackageDefinitions object.
func NewPackageDefinitions(name, pkgPath string) *PackageDefinitions {
	return domain.NewPackageDefinitions(name, pkgPath)
}

// Re-export utility functions for backward compatibility

// IsGolangPrimitiveType checks if a type is a Go primitive type.
func IsGolangPrimitiveType(typeName string) bool {
	return domain.IsGolangPrimitiveType(typeName)
}

// EvaluateEscapedChar parse escaped character
func EvaluateEscapedChar(text string) rune {
	return domain.EvaluateEscapedChar(text)
}

// EvaluateEscapedString parse escaped characters in string
func EvaluateEscapedString(text string) string {
	return domain.EvaluateEscapedString(text)
}

// EvaluateDataConversion evaluate the type a explicit type conversion
func EvaluateDataConversion(x interface{}, typeName string) interface{} {
	return domain.EvaluateDataConversion(x, typeName)
}

// EvaluateUnary evaluate the type and value of a unary expression
func EvaluateUnary(x interface{}, operator token.Token, xtype ast.Expr) (interface{}, ast.Expr) {
	return domain.EvaluateUnary(x, operator, xtype)
}

// EvaluateBinary evaluate the type and value of a binary expression
func EvaluateBinary(x, y interface{}, operator token.Token, xtype, ytype ast.Expr) (interface{}, ast.Expr) {
	return domain.EvaluateBinary(x, y, operator, xtype, ytype)
}

// TransToValidPrimitiveSchema transfer golang basic type to swagger schema with format considered.
func TransToValidPrimitiveSchema(typeName string) *spec.Schema {
	return domain.TransToValidPrimitiveSchema(typeName)
}
