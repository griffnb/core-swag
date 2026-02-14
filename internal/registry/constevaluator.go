// Package registry - constant expression evaluation.
package registry

import (
	"go/ast"
	"go/token"
	"reflect"
	"strconv"

	"github.com/griffnb/core-swag/internal/domain"
)

// evaluateConstValue evaluates a const expression value.
func (s *Service) evaluateConstValueHelper(
	pkg *domain.PackageDefinitions,
	file *ast.File,
	iota int,
	expr ast.Expr,
	recursiveStack map[string]struct{},
) (interface{}, ast.Expr) {
	switch valueExpr := expr.(type) {
	case *ast.Ident:
		if valueExpr.Name == "iota" {
			return iota, nil
		}
		if pkg.ConstTable != nil {
			if cv, ok := pkg.ConstTable[valueExpr.Name]; ok {
				return s.EvaluateConstValue(pkg, cv, recursiveStack)
			}
		}
	case *ast.SelectorExpr:
		pkgIdent, ok := valueExpr.X.(*ast.Ident)
		if !ok {
			return nil, nil
		}
		return s.EvaluateConstValueByName(file, pkgIdent.Name, valueExpr.Sel.Name, recursiveStack)
	case *ast.BasicLit:
		switch valueExpr.Kind {
		case token.INT:
			if x, err := strconv.ParseInt(valueExpr.Value, 0, 64); err == nil {
				return int(x), nil
			} else if x, err := strconv.ParseUint(valueExpr.Value, 0, 64); err == nil {
				return x, nil
			} else {
				panic(err)
			}
		case token.STRING:
			if valueExpr.Value[0] == '`' {
				return valueExpr.Value[1 : len(valueExpr.Value)-1], nil
			}
			return domain.EvaluateEscapedString(valueExpr.Value[1 : len(valueExpr.Value)-1]), nil
		case token.CHAR:
			return domain.EvaluateEscapedChar(valueExpr.Value[1 : len(valueExpr.Value)-1]), nil
		}
	case *ast.UnaryExpr:
		x, evalType := s.evaluateConstValueHelper(pkg, file, iota, valueExpr.X, recursiveStack)
		if x == nil {
			return x, evalType
		}
		return domain.EvaluateUnary(x, valueExpr.Op, evalType)
	case *ast.BinaryExpr:
		x, evalTypex := s.evaluateConstValueHelper(pkg, file, iota, valueExpr.X, recursiveStack)
		y, evalTypey := s.evaluateConstValueHelper(pkg, file, iota, valueExpr.Y, recursiveStack)
		if x == nil || y == nil {
			return nil, nil
		}
		return domain.EvaluateBinary(x, y, valueExpr.Op, evalTypex, evalTypey)
	case *ast.ParenExpr:
		return s.evaluateConstValueHelper(pkg, file, iota, valueExpr.X, recursiveStack)
	case *ast.CallExpr:
		if len(valueExpr.Args) != 1 {
			return nil, nil
		}
		arg := valueExpr.Args[0]
		if ident, ok := valueExpr.Fun.(*ast.Ident); ok {
			name := ident.Name
			if name == "uintptr" {
				name = "uint"
			}
			value, _ := s.evaluateConstValueHelper(pkg, file, iota, arg, recursiveStack)
			if domain.IsGolangPrimitiveType(name) {
				value = domain.EvaluateDataConversion(value, name)
				return value, nil
			} else if name == "len" {
				return reflect.ValueOf(value).Len(), nil
			}
			typeDef := s.FindTypeSpec(name, file)
			if typeDef == nil {
				return nil, nil
			}
			return value, valueExpr.Fun
		} else if selector, ok := valueExpr.Fun.(*ast.SelectorExpr); ok {
			typeDef := s.FindTypeSpec(fullTypeName(selector.X.(*ast.Ident).Name, selector.Sel.Name), file)
			if typeDef == nil {
				return nil, nil
			}
			return arg, typeDef.TypeSpec.Type
		}
	}
	return nil, nil
}
