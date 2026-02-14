// Package registry - enum and constant handling functionality.
package registry

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/swaggo/swag/console"
	"github.com/swaggo/swag/internal/domain"
)

func (s *Service) collectConstVariables(astFile *ast.File, packagePath string, generalDeclaration *ast.GenDecl) {
	pkg, ok := s.packages[packagePath]
	if !ok {
		pkg = domain.NewPackageDefinitions(astFile.Name.Name, packagePath)
		s.packages[packagePath] = pkg
	}

	var lastValueSpec *ast.ValueSpec
	for _, astSpec := range generalDeclaration.Specs {
		valueSpec, ok := astSpec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if len(valueSpec.Names) == 1 && len(valueSpec.Values) == 1 {
			lastValueSpec = valueSpec
		} else if len(valueSpec.Names) == 1 && len(valueSpec.Values) == 0 && valueSpec.Type == nil && lastValueSpec != nil {
			valueSpec.Type = lastValueSpec.Type
			valueSpec.Values = lastValueSpec.Values
		}
		pkg.AddConst(astFile, valueSpec)
	}
}

func (s *Service) evaluateAllConstVariables() {
	for _, pkg := range s.packages {
		for _, constVar := range pkg.OrderedConst {
			s.EvaluateConstValue(pkg, constVar, nil)
		}
	}
}

// EvaluateConstValue evaluates a const variable.
func (s *Service) EvaluateConstValue(
	pkg *domain.PackageDefinitions,
	cv *domain.ConstVariable,
	recursiveStack map[string]struct{},
) (interface{}, ast.Expr) {
	if pkg.Package != nil {
		obj := pkg.Package.Types.Scope().Lookup(cv.Name.Name)
		if obj != nil {
			if constObj, ok := obj.(*types.Const); ok {
				cv.Value = constant.Val(constObj.Val())
				if cv.Type == nil {
					cv.Type = tryParseTypeFromPackage(pkg.Package, constObj)
				}
				return cv.Value, cv.Type
			}
		}
	}
	if expr, ok := cv.Value.(ast.Expr); ok {
		defer func() {
			if err := recover(); err != nil {
				if fi, ok := s.files[cv.File]; ok {
					pos := fi.FileSet.Position(cv.Name.NamePos)
					console.Logger.Debug(
						"warning: failed to evaluate const %s at %s:%d:%d, %v",
						cv.Name.Name,
						fi.Path,
						pos.Line,
						pos.Column,
						err,
					)
				}
			}
		}()
		if recursiveStack == nil {
			recursiveStack = make(map[string]struct{})
		}
		fullConstName := fullTypeName(pkg.Path, cv.Name.Name)
		if _, ok = recursiveStack[fullConstName]; ok {
			return nil, nil
		}
		recursiveStack[fullConstName] = struct{}{}

		value, evalType := s.evaluateConstValueHelper(pkg, cv.File, cv.Name.Obj.Data.(int), expr, recursiveStack)
		if cv.Type == nil && evalType != nil {
			cv.Type = evalType
		}
		if value != nil {
			cv.Value = value
		}
		return value, cv.Type
	}
	return cv.Value, cv.Type
}

// EvaluateConstValueByName evaluates a const variable by name.
func (s *Service) EvaluateConstValueByName(
	file *ast.File,
	pkgName, constVariableName string,
	recursiveStack map[string]struct{},
) (interface{}, ast.Expr) {
	matchedPkgPaths, externalPkgPaths := s.findPackagePathFromImports(pkgName, file)
	for _, pkgPath := range matchedPkgPaths {
		if pkg, ok := s.packages[pkgPath]; ok {
			if cv, ok := pkg.ConstTable[constVariableName]; ok {
				return s.EvaluateConstValue(pkg, cv, recursiveStack)
			}
		}
	}
	if s.parseDependency > 0 {
		for _, pkgPath := range externalPkgPaths {
			if err := s.loadExternalPackage(pkgPath); err == nil {
				if pkg, ok := s.packages[pkgPath]; ok {
					if cv, ok := pkg.ConstTable[constVariableName]; ok {
						return s.EvaluateConstValue(pkg, cv, recursiveStack)
					}
				}
			}
		}
	}
	return nil, nil
}

func (s *Service) collectConstEnums(parsedSchemas map[*domain.TypeSpecDef]*domain.Schema) {
	for _, pkg := range s.packages {
		for _, constVar := range pkg.OrderedConst {
			if constVar.Type == nil {
				continue
			}
			ident, ok := constVar.Type.(*ast.Ident)
			if !ok || domain.IsGolangPrimitiveType(ident.Name) {
				continue
			}
			typeDef, ok := pkg.TypeDefinitions[ident.Name]
			if !ok {
				continue
			}

			// delete it from parsed schemas, and will parse it again
			if _, ok = parsedSchemas[typeDef]; ok {
				delete(parsedSchemas, typeDef)
			}

			if typeDef.Enums == nil {
				typeDef.Enums = make([]domain.EnumValue, 0)
			}

			if _, ok = constVar.Value.(ast.Expr); ok {
				continue
			}

			// EnumValue has unexported key field, so we cannot set it directly
			// We need to use a workaround or export the field in swag package
			// For now, skip setting the key field
			enumValue := domain.EnumValue{
				Value:   constVar.Value,
				Comment: commentWithoutNameOverride(constVar.Comment),
			}
			typeDef.Enums = append(typeDef.Enums, enumValue)
		}
	}
}

