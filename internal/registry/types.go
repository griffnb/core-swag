// Package registry - type parsing and registration functionality.
package registry

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/swaggo/swag/console"
	"github.com/swaggo/swag/internal/domain"
	"golang.org/x/tools/go/packages"
)

// ParseTypes parses types from registered files.
func (s *Service) ParseTypes() (map[*domain.TypeSpecDef]*domain.Schema, error) {
	parsedSchemas := make(map[*domain.TypeSpecDef]*domain.Schema)
	for astFile, info := range s.files {
		s.parseTypesFromFile(astFile, info.PackagePath, parsedSchemas)
		s.parseFunctionScopedTypesFromFile(astFile, info.PackagePath, parsedSchemas)
	}
	s.removeAllNotUniqueTypes()
	s.evaluateAllConstVariables()
	s.collectConstEnums(parsedSchemas)
	return parsedSchemas, nil
}

func (s *Service) parseTypesFromFile(astFile *ast.File, packagePath string, parsedSchemas map[*domain.TypeSpecDef]*domain.Schema) {
	for _, astDeclaration := range astFile.Decls {
		generalDeclaration, ok := astDeclaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		if generalDeclaration.Tok == token.TYPE {
			for _, astSpec := range generalDeclaration.Specs {
				if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
					typeSpecDef := &domain.TypeSpecDef{
						PkgPath:  packagePath,
						File:     astFile,
						TypeSpec: typeSpec,
					}

					if idt, ok := typeSpec.Type.(*ast.Ident); ok && domain.IsGolangPrimitiveType(idt.Name) && parsedSchemas != nil {
						parsedSchemas[typeSpecDef] = &domain.Schema{
							PkgPath: typeSpecDef.PkgPath,
							Name:    astFile.Name.Name,
							Schema:  domain.TransToValidPrimitiveSchema(idt.Name),
						}
					}

					if s.uniqueDefinitions == nil {
						s.uniqueDefinitions = make(map[string]*domain.TypeSpecDef)
					}

					fullName := typeSpecDef.TypeName()

					anotherTypeDef, ok := s.uniqueDefinitions[fullName]
					if ok {
						if anotherTypeDef == nil {
							typeSpecDef.NotUnique = true
							fullName = typeSpecDef.TypeName()
							s.uniqueDefinitions[fullName] = typeSpecDef
						} else if typeSpecDef.PkgPath != anotherTypeDef.PkgPath {
							s.uniqueDefinitions[fullName] = nil
							anotherTypeDef.NotUnique = true
							s.uniqueDefinitions[anotherTypeDef.TypeName()] = anotherTypeDef
							anotherTypeDef.SetSchemaName()

							typeSpecDef.NotUnique = true
							fullName = typeSpecDef.TypeName()
							s.uniqueDefinitions[fullName] = typeSpecDef
						}
					} else {
						s.uniqueDefinitions[fullName] = typeSpecDef
					}

					typeSpecDef.SetSchemaName()

					if s.packages[typeSpecDef.PkgPath] == nil {
						s.packages[typeSpecDef.PkgPath] = domain.NewPackageDefinitions(
							astFile.Name.Name,
							typeSpecDef.PkgPath,
						).AddTypeSpec(typeSpecDef.Name(), typeSpecDef)
					} else if _, ok = s.packages[typeSpecDef.PkgPath].TypeDefinitions[typeSpecDef.Name()]; !ok {
						s.packages[typeSpecDef.PkgPath].AddTypeSpec(typeSpecDef.Name(), typeSpecDef)
					}
				}
			}
		} else if generalDeclaration.Tok == token.CONST {
			// collect consts
			s.collectConstVariables(astFile, packagePath, generalDeclaration)
		}
	}
}

func (s *Service) parseFunctionScopedTypesFromFile(astFile *ast.File, packagePath string, parsedSchemas map[*domain.TypeSpecDef]*domain.Schema) {
	for _, astDeclaration := range astFile.Decls {
		funcDeclaration, ok := astDeclaration.(*ast.FuncDecl)
		if ok && funcDeclaration.Body != nil {
			functionScopedTypes := make(map[string]*domain.TypeSpecDef)
			for _, stmt := range funcDeclaration.Body.List {
				if declStmt, ok := (stmt).(*ast.DeclStmt); ok {
					if genDecl, ok := (declStmt.Decl).(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
						for _, astSpec := range genDecl.Specs {
							if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
								typeSpecDef := &domain.TypeSpecDef{
									PkgPath:    packagePath,
									File:       astFile,
									TypeSpec:   typeSpec,
									ParentSpec: astDeclaration,
								}

								if idt, ok := typeSpec.Type.(*ast.Ident); ok && domain.IsGolangPrimitiveType(idt.Name) && parsedSchemas != nil {
									parsedSchemas[typeSpecDef] = &domain.Schema{
										PkgPath: typeSpecDef.PkgPath,
										Name:    astFile.Name.Name,
										Schema:  domain.TransToValidPrimitiveSchema(idt.Name),
									}
								}

								fullName := typeSpecDef.TypeName()
								if structType, ok := typeSpecDef.TypeSpec.Type.(*ast.StructType); ok {
									for _, field := range structType.Fields.List {
										var idt *ast.Ident
										var ok bool
										switch field.Type.(type) {
										case *ast.Ident:
											idt, ok = field.Type.(*ast.Ident)
										case *ast.StarExpr:
											idt, ok = field.Type.(*ast.StarExpr).X.(*ast.Ident)
										case *ast.ArrayType:
											idt, ok = field.Type.(*ast.ArrayType).Elt.(*ast.Ident)
										}
										if ok && !domain.IsGolangPrimitiveType(idt.Name) {
											if functype, ok := functionScopedTypes[idt.Name]; ok {
												idt.Name = functype.TypeName()
											}
										}
									}
								}

								if s.uniqueDefinitions == nil {
									s.uniqueDefinitions = make(map[string]*domain.TypeSpecDef)
								}

								anotherTypeDef, ok := s.uniqueDefinitions[fullName]
								if ok {
									if anotherTypeDef == nil {
										typeSpecDef.NotUnique = true
										fullName = typeSpecDef.TypeName()
										s.uniqueDefinitions[fullName] = typeSpecDef
									} else if typeSpecDef.PkgPath != anotherTypeDef.PkgPath {
										s.uniqueDefinitions[fullName] = nil
										anotherTypeDef.NotUnique = true
										s.uniqueDefinitions[anotherTypeDef.TypeName()] = anotherTypeDef
										anotherTypeDef.SetSchemaName()

										typeSpecDef.NotUnique = true
										fullName = typeSpecDef.TypeName()
										s.uniqueDefinitions[fullName] = typeSpecDef
									}
								} else {
									s.uniqueDefinitions[fullName] = typeSpecDef
									functionScopedTypes[typeSpec.Name.Name] = typeSpecDef
								}

								typeSpecDef.SetSchemaName()

								if s.packages[typeSpecDef.PkgPath] == nil {
									s.packages[typeSpecDef.PkgPath] = domain.NewPackageDefinitions(
										astFile.Name.Name,
										typeSpecDef.PkgPath,
									).AddTypeSpec(fullName, typeSpecDef)
								} else if _, ok = s.packages[typeSpecDef.PkgPath].TypeDefinitions[fullName]; !ok {
									s.packages[typeSpecDef.PkgPath].AddTypeSpec(fullName, typeSpecDef)
								}
							}
						}
					}
				}
			}
		}
	}
}

func (s *Service) removeAllNotUniqueTypes() {
	for key, ud := range s.uniqueDefinitions {
		if ud == nil {
			delete(s.uniqueDefinitions, key)
		}
	}
}

// FindTypeSpec finds TypeSpecDef by type name.
func (s *Service) FindTypeSpec(typeName string, file *ast.File) *domain.TypeSpecDef {
	if domain.IsGolangPrimitiveType(typeName) {
		return nil
	}

	if file == nil { // for test
		return s.uniqueDefinitions[typeName]
	}

	parts := strings.Split(strings.Split(typeName, "[")[0], ".")
	if len(parts) > 1 {
		pkgPaths, externalPkgPaths := s.findPackagePathFromImports(parts[0], file)
		if len(externalPkgPaths) == 0 || s.parseDependency == domain.ParseNone {
			typeDef, ok := s.uniqueDefinitions[typeName]
			if ok {
				return typeDef
			}
		}
		typeDef := s.findTypeSpecFromPackagePaths(pkgPaths, externalPkgPaths, parts[1])
		return s.parametrizeGenericType(file, typeDef, typeName)
	}

	typeDef, ok := s.uniqueDefinitions[fullTypeName(file.Name.Name, typeName)]
	if ok {
		return typeDef
	}

	name := parts[0]
	typeDef, ok = s.uniqueDefinitions[fullTypeName(file.Name.Name, name)]
	if !ok {
		pkgPaths, externalPkgPaths := s.findPackagePathFromImports("", file)
		typeDef = s.findTypeSpecFromPackagePaths(pkgPaths, externalPkgPaths, name)
	}

	if typeDef != nil {
		return s.parametrizeGenericType(file, typeDef, typeName)
	}

	// in case that comment //@name renamed the type with a name without a dot
	for k, v := range s.uniqueDefinitions {
		if v == nil {
			if s.debug != nil {
				s.debug.Printf("%s TypeSpecDef is nil", k)
			}
			continue
		}
		if v.SchemaName == typeName {
			return v
		}
	}

	return nil
}

func (s *Service) findTypeSpec(pkgPath string, typeName string) *domain.TypeSpecDef {
	if s.packages == nil {
		return nil
	}

	pd, found := s.packages[pkgPath]
	if found {
		typeSpec, ok := pd.TypeDefinitions[typeName]
		if ok {
			return typeSpec
		}
	}

	return nil
}

// CheckTypeSpec checks if a TypeSpecDef has MarshalJSON method.
func (s *Service) CheckTypeSpec(typeSpecDef *domain.TypeSpecDef) {
	if typeSpecDef == nil {
		return
	}

	packageDefinition := s.packages[typeSpecDef.PkgPath]
	if packageDefinition == nil {
		return
	}
	pkg := packageDefinition.Package
	if pkg == nil {
		return
	}
	obj := pkg.TypesInfo.ObjectOf(typeSpecDef.TypeSpec.Name)
	if obj == nil {
		obj = findGenericTypeFromPackage(pkg, typeSpecDef.TypeSpec.Name.Pos())
	}
	if obj == nil {
		if s.debug != nil {
			s.debug.Printf("warning: %s TypeSpecDef is nil", typeSpecDef.TypeSpec.Name.Name)
		}
		return
	}
	s.checkJSONMarshal(pkg, obj)
}

func (s *Service) checkJSONMarshal(pkg *packages.Package, obj types.Object) {
	methodSet := types.NewMethodSet(obj.Type())
	method := methodSet.Lookup(pkg.Types, "MarshalJSON")
	if method != nil {
		console.Logger.Debug("warning: %s.%s has MarshalJSON method, may need special handling", pkg.PkgPath, obj.Name())
	}
}

func findGenericTypeFromPackage(pkg *packages.Package, pos token.Pos) types.Object {
	file := findFileInPackageByPos(pkg, pos)
	if file == nil {
		return nil
	}
	var found ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil || found != nil {
			return false
		}
		if node.Pos() == pos {
			found = node
			return false
		}
		return true
	})
	typeSpec, _ := found.(*ast.TypeSpec)
	if typeSpec == nil {
		return nil
	}
	return pkg.TypesInfo.ObjectOf(typeSpec.Name)
}

func findFileInPackageByPos(pkg *packages.Package, pos token.Pos) *ast.File {
	for _, file := range pkg.Syntax {
		if file.Pos() <= pos && pos <= file.End() {
			return file
		}
	}
	return nil
}
