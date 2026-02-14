// Package registry - helper functions for type name construction and comment parsing.
package registry

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

func fullTypeName(parts ...string) string {
	return strings.Join(parts, ".")
}

func commentWithoutNameOverride(comment string) string {
	commentStr := strings.TrimSpace(strings.TrimLeft(comment, "/"))
	if strings.HasPrefix(commentStr, "@name") {
		parts := strings.SplitN(commentStr, " ", 3)
		if len(parts) > 2 {
			return strings.TrimSpace(parts[2])
		}
		return ""
	}
	return comment
}

func tryParseTypeFromPackage(pkg *packages.Package, constObj *types.Const) ast.Expr {
	// This function is not needed for basic functionality
	// It's a helper for const type inference
	return nil
}
