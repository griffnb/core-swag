// Package domain contains core domain types shared across the swag application.
// These types represent fundamental domain objects like type specifications,
// AST file information, and package definitions.
package domain

import (
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/loader"
	"golang.org/x/tools/go/packages"
)

// Schema parsed schema.
type Schema struct {
	*spec.Schema        //
	PkgPath      string // package import path used to rename Name of a definition int case of conflict
	Name         string // Name in definitions
}

// TypeSpecDef the whole information of a typeSpec.
type TypeSpecDef struct {
	// ast file where TypeSpec is
	File *ast.File

	// the TypeSpec of this type definition
	TypeSpec *ast.TypeSpec

	Enums []EnumValue

	// path of package starting from under ${GOPATH}/src or from module path in go.mod
	PkgPath    string
	ParentSpec ast.Decl

	SchemaName string

	NotUnique bool
}

// Name the name of the typeSpec.
func (t *TypeSpecDef) Name() string {
	if t.TypeSpec != nil && t.TypeSpec.Name != nil {
		return t.TypeSpec.Name.Name
	}

	return ""
}

// TypeName the type name of the typeSpec.
func (t *TypeSpecDef) TypeName() string {
	if ignoreNameOverride(t.TypeSpec.Name.Name) {
		return t.TypeSpec.Name.Name[1:]
	}

	var names []string
	if t.NotUnique {
		pkgPath := strings.Map(func(r rune) rune {
			if r == '\\' || r == '/' || r == '.' {
				return '_'
			}
			return r
		}, t.PkgPath)
		names = append(names, pkgPath)
	} else if t.File != nil {
		names = append(names, t.File.Name.Name)
	}
	if parentFun, ok := (t.ParentSpec).(*ast.FuncDecl); ok && parentFun != nil {
		names = append(names, parentFun.Name.Name)
	}
	names = append(names, t.TypeSpec.Name.Name)
	return fullTypeName(names...)
}

// SimpleTypeName returns a simplified type name (package.Type format).
func (t *TypeSpecDef) SimpleTypeName() string {
	if ignoreNameOverride(t.TypeSpec.Name.Name) {
		return t.TypeSpec.Name.Name[1:]
	}

	var names []string
	// Only use package name, not full path
	if t.File != nil {
		names = append(names, t.File.Name.Name)
	}
	if parentFun, ok := (t.ParentSpec).(*ast.FuncDecl); ok && parentFun != nil {
		names = append(names, parentFun.Name.Name)
	}
	names = append(names, t.TypeSpec.Name.Name)
	return fullTypeName(names...)
}

// FullPath return the full path of the typeSpec.
func (t *TypeSpecDef) FullPath() string {
	return t.PkgPath + "." + t.Name()
}

func (t *TypeSpecDef) Alias() string {
	return nameOverride(t.TypeSpec.Comment)
}

func (t *TypeSpecDef) SetSchemaName() {
	if alias := t.Alias(); alias != "" {
		t.SchemaName = alias
		return
	}

	t.SchemaName = t.TypeName()
}

// SetSchemaNameSimple sets schema name using simplified naming (package.Type).
func (t *TypeSpecDef) SetSchemaNameSimple() {
	if alias := t.Alias(); alias != "" {
		t.SchemaName = alias
		return
	}

	t.SchemaName = t.SimpleTypeName()
}

// AstFileInfo information of an ast.File.
type AstFileInfo struct {
	// FileSet the FileSet object which is used to parse this go source file
	FileSet *token.FileSet

	// File ast.File
	File *ast.File

	// Path the path of the ast.File
	Path string

	// PackagePath package import path of the ast.File
	PackagePath string

	// ParseFlag determine what to parse
	ParseFlag ParseFlag
}

// PackageDefinitions files and definition in a package.
type PackageDefinitions struct {
	// files in this package, map key is file's relative path starting package path
	Files map[string]*ast.File

	// definitions in this package, map key is typeName
	TypeDefinitions map[string]*TypeSpecDef

	// const variables in this package, map key is the name
	ConstTable map[string]*ConstVariable

	// const variables in order in this package
	OrderedConst []*ConstVariable

	// package name
	Name string

	// package path
	Path string

	Package *packages.Package
}

// ConstVariableGlobalEvaluator an interface used to evaluate enums across packages
type ConstVariableGlobalEvaluator interface {
	EvaluateConstValue(pkg *PackageDefinitions, cv *ConstVariable, recursiveStack map[string]struct{}) (interface{}, ast.Expr)
	EvaluateConstValueByName(file *ast.File, pkgPath, constVariableName string, recursiveStack map[string]struct{}) (interface{}, ast.Expr)
	FindTypeSpec(typeName string, file *ast.File) *TypeSpecDef
}

// NewPackageDefinitions new a PackageDefinitions object
func NewPackageDefinitions(name, pkgPath string) *PackageDefinitions {
	return &PackageDefinitions{
		Name:            name,
		Path:            pkgPath,
		Files:           make(map[string]*ast.File),
		TypeDefinitions: make(map[string]*TypeSpecDef),
		ConstTable:      make(map[string]*ConstVariable),
	}
}

// AddFile add a file
func (pkg *PackageDefinitions) AddFile(pkgPath string, file *ast.File) *PackageDefinitions {
	pkg.Files[pkgPath] = file
	return pkg
}

// AddTypeSpec add a type spec.
func (pkg *PackageDefinitions) AddTypeSpec(name string, typeSpec *TypeSpecDef) *PackageDefinitions {
	pkg.TypeDefinitions[name] = typeSpec
	return pkg
}

// AddConst add a const variable.
func (pkg *PackageDefinitions) AddConst(astFile *ast.File, valueSpec *ast.ValueSpec) *PackageDefinitions {
	for i := 0; i < len(valueSpec.Names) && i < len(valueSpec.Values); i++ {
		variable := &ConstVariable{
			Name:  valueSpec.Names[i],
			Type:  valueSpec.Type,
			Value: valueSpec.Values[i],
			File:  astFile,
		}
		//take the nearest line as comment from comment list or doc list. comment list first.
		if valueSpec.Comment != nil && len(valueSpec.Comment.List) > 0 {
			variable.Comment = valueSpec.Comment.List[0].Text
		} else if valueSpec.Doc != nil && len(valueSpec.Doc.List) > 0 {
			variable.Comment = valueSpec.Doc.List[len(valueSpec.Doc.List)-1].Text
		}
		pkg.ConstTable[valueSpec.Names[i].Name] = variable
		pkg.OrderedConst = append(pkg.OrderedConst, variable)
	}
	return pkg
}

func (pkg *PackageDefinitions) EvaluateConstValue(file *ast.File, iota int, expr ast.Expr, globalEvaluator ConstVariableGlobalEvaluator, recursiveStack map[string]struct{}) (interface{}, ast.Expr) {
	switch valueExpr := expr.(type) {
	case *ast.Ident:
		if valueExpr.Name == "iota" {
			return iota, nil
		}
		if pkg.ConstTable != nil {
			if cv, ok := pkg.ConstTable[valueExpr.Name]; ok {
				return globalEvaluator.EvaluateConstValue(pkg, cv, recursiveStack)
			}
		}
	case *ast.SelectorExpr:
		pkgIdent, ok := valueExpr.X.(*ast.Ident)
		if !ok {
			return nil, nil
		}
		return globalEvaluator.EvaluateConstValueByName(file, pkgIdent.Name, valueExpr.Sel.Name, recursiveStack)
	case *ast.BasicLit:
		switch valueExpr.Kind {
		case token.INT:
			//a basic literal integer is int type in default, or must have an explicit converting type in front
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
			return EvaluateEscapedString(valueExpr.Value[1 : len(valueExpr.Value)-1]), nil
		case token.CHAR:
			return EvaluateEscapedChar(valueExpr.Value[1 : len(valueExpr.Value)-1]), nil
		}
	case *ast.UnaryExpr:
		x, evalType := pkg.EvaluateConstValue(file, iota, valueExpr.X, globalEvaluator, recursiveStack)
		if x == nil {
			return x, evalType
		}
		return EvaluateUnary(x, valueExpr.Op, evalType)
	case *ast.BinaryExpr:
		x, evalTypex := pkg.EvaluateConstValue(file, iota, valueExpr.X, globalEvaluator, recursiveStack)
		y, evalTypey := pkg.EvaluateConstValue(file, iota, valueExpr.Y, globalEvaluator, recursiveStack)
		if x == nil || y == nil {
			return nil, nil
		}
		return EvaluateBinary(x, y, valueExpr.Op, evalTypex, evalTypey)
	case *ast.ParenExpr:
		return pkg.EvaluateConstValue(file, iota, valueExpr.X, globalEvaluator, recursiveStack)
	case *ast.CallExpr:
		//data conversion
		if len(valueExpr.Args) != 1 {
			return nil, nil
		}
		arg := valueExpr.Args[0]
		if ident, ok := valueExpr.Fun.(*ast.Ident); ok {
			name := ident.Name
			if name == "uintptr" {
				name = "uint"
			}
			value, _ := pkg.EvaluateConstValue(file, iota, arg, globalEvaluator, recursiveStack)
			if IsGolangPrimitiveType(name) {
				value = EvaluateDataConversion(value, name)
				return value, nil
			} else if name == "len" {
				return reflect.ValueOf(value).Len(), nil
			}
			typeDef := globalEvaluator.FindTypeSpec(name, file)
			if typeDef == nil {
				return nil, nil
			}
			return value, valueExpr.Fun
		} else if selector, ok := valueExpr.Fun.(*ast.SelectorExpr); ok {
			typeDef := globalEvaluator.FindTypeSpec(fullTypeName(selector.X.(*ast.Ident).Name, selector.Sel.Name), file)
			if typeDef == nil {
				return nil, nil
			}
			return arg, typeDef.TypeSpec.Type
		}
	}
	return nil, nil
}

// ConstVariable a model to record a const variable
type ConstVariable struct {
	Name    *ast.Ident
	Type    ast.Expr
	Value   interface{}
	Comment string
	File    *ast.File
	Pkg     *PackageDefinitions
}

// VariableName gets the name for this const variable, taking into account comment overrides.
func (cv *ConstVariable) VariableName() string {
	if ignoreNameOverride(cv.Name.Name) {
		return cv.Name.Name[1:]
	}

	if overriddenName := cv.nameOverride(); overriddenName != "" {
		return overriddenName
	}

	return cv.Name.Name
}

func (cv *ConstVariable) nameOverride() string {
	if len(cv.Comment) == 0 {
		return ""
	}

	comment := strings.TrimSpace(strings.TrimLeft(cv.Comment, "/"))
	texts := overrideNameRegex.FindStringSubmatch(comment)
	if len(texts) > 1 {
		return texts[1]
	}
	return ""
}

const (
	// EnumVarNamesExtension x-enum-varnames
	EnumVarNamesExtension     = "x-enum-varnames"
	// EnumCommentsExtension x-enum-comments
	EnumCommentsExtension     = "x-enum-comments"
	// EnumDescriptionsExtension x-enum-descriptions
	EnumDescriptionsExtension = "x-enum-descriptions"
)

// EnumValue a model to record an enum consts variable
type EnumValue struct {
	Key     string
	Value   interface{}
	Comment string
}

// ParseFlag is an alias for loader.ParseFlag
type ParseFlag = loader.ParseFlag

const (
	// ParseNone parse nothing
	ParseNone = loader.ParseNone
	// ParseModels parse models
	ParseModels = loader.ParseModels
	// ParseOperations parse operations
	ParseOperations = loader.ParseOperations
	// ParseAll parse everything
	ParseAll = loader.ParseAll
)

var escapedChars = map[uint8]uint8{
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\v',
	'\\': '\\',
	'"':  '"',
}

// EvaluateEscapedChar parse escaped character
func EvaluateEscapedChar(text string) rune {
	if len(text) == 1 {
		return rune(text[0])
	}

	if len(text) == 2 && text[0] == '\\' {
		return rune(escapedChars[text[1]])
	}

	if len(text) == 6 && text[0:2] == "\\u" {
		n, err := strconv.ParseInt(text[2:], 16, 32)
		if err == nil {
			return rune(n)
		}
	}

	return 0
}

// EvaluateEscapedString parse escaped characters in string
func EvaluateEscapedString(text string) string {
	if !strings.ContainsRune(text, '\\') {
		return text
	}
	result := make([]byte, 0, len(text))
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' {
			i++
			if text[i] == 'u' {
				i++
				char, err := strconv.ParseInt(text[i:i+4], 16, 32)
				if err == nil {
					result = utf8.AppendRune(result, rune(char))
				}
				i += 3
			} else if c, ok := escapedChars[text[i]]; ok {
				result = append(result, c)
			}
		} else {
			result = append(result, text[i])
		}
	}
	return string(result)
}

// EvaluateDataConversion evaluate the type a explicit type conversion
func EvaluateDataConversion(x interface{}, typeName string) interface{} {
	switch value := x.(type) {
	case int:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case uint:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case int8:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case uint8:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case int16:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case uint16:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case int32:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		case "string":
			return string(value)
		}
	case uint32:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case int64:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case uint64:
		switch typeName {
		case "int":
			return int(value)
		case "byte":
			return byte(value)
		case "int8":
			return int8(value)
		case "int16":
			return int16(value)
		case "int32":
			return int32(value)
		case "int64":
			return int64(value)
		case "uint":
			return uint(value)
		case "uint8":
			return uint8(value)
		case "uint16":
			return uint16(value)
		case "uint32":
			return uint32(value)
		case "uint64":
			return uint64(value)
		case "rune":
			return rune(value)
		}
	case string:
		switch typeName {
		case "string":
			return value
		}
	}
	return nil
}

// EvaluateUnary evaluate the type and value of a unary expression
func EvaluateUnary(x interface{}, operator token.Token, xtype ast.Expr) (interface{}, ast.Expr) {
	switch operator {
	case token.SUB:
		switch value := x.(type) {
		case int:
			return -value, xtype
		case int8:
			return -value, xtype
		case int16:
			return -value, xtype
		case int32:
			return -value, xtype
		case int64:
			return -value, xtype
		}
	case token.XOR:
		switch value := x.(type) {
		case int:
			return ^value, xtype
		case int8:
			return ^value, xtype
		case int16:
			return ^value, xtype
		case int32:
			return ^value, xtype
		case int64:
			return ^value, xtype
		case uint:
			return ^value, xtype
		case uint8:
			return ^value, xtype
		case uint16:
			return ^value, xtype
		case uint32:
			return ^value, xtype
		case uint64:
			return ^value, xtype
		}
	}
	return nil, nil
}

// EvaluateBinary evaluate the type and value of a binary expression
func EvaluateBinary(x, y interface{}, operator token.Token, xtype, ytype ast.Expr) (interface{}, ast.Expr) {
	if operator == token.SHR || operator == token.SHL {
		var rightOperand uint64
		yValue := reflect.ValueOf(y)
		if yValue.CanUint() {
			rightOperand = yValue.Uint()
		} else if yValue.CanInt() {
			rightOperand = uint64(yValue.Int())
		}

		switch operator {
		case token.SHL:
			switch xValue := x.(type) {
			case int:
				return xValue << rightOperand, xtype
			case int8:
				return xValue << rightOperand, xtype
			case int16:
				return xValue << rightOperand, xtype
			case int32:
				return xValue << rightOperand, xtype
			case int64:
				return xValue << rightOperand, xtype
			case uint:
				return xValue << rightOperand, xtype
			case uint8:
				return xValue << rightOperand, xtype
			case uint16:
				return xValue << rightOperand, xtype
			case uint32:
				return xValue << rightOperand, xtype
			case uint64:
				return xValue << rightOperand, xtype
			}
		case token.SHR:
			switch xValue := x.(type) {
			case int:
				return xValue >> rightOperand, xtype
			case int8:
				return xValue >> rightOperand, xtype
			case int16:
				return xValue >> rightOperand, xtype
			case int32:
				return xValue >> rightOperand, xtype
			case int64:
				return xValue >> rightOperand, xtype
			case uint:
				return xValue >> rightOperand, xtype
			case uint8:
				return xValue >> rightOperand, xtype
			case uint16:
				return xValue >> rightOperand, xtype
			case uint32:
				return xValue >> rightOperand, xtype
			case uint64:
				return xValue >> rightOperand, xtype
			}
		}
		return nil, nil
	}

	evalType := xtype
	if evalType == nil {
		evalType = ytype
	}

	xValue := reflect.ValueOf(x)
	yValue := reflect.ValueOf(y)
	if xValue.Kind() == reflect.String && yValue.Kind() == reflect.String {
		return xValue.String() + yValue.String(), evalType
	}

	var targetValue reflect.Value
	if xValue.Kind() != reflect.Int {
		targetValue = reflect.New(xValue.Type()).Elem()
	} else {
		targetValue = reflect.New(yValue.Type()).Elem()
	}

	switch operator {
	case token.ADD:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() + yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() + yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) + yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() + uint64(yValue.Int()))
		}
	case token.SUB:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() - yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() - yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) - yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() - uint64(yValue.Int()))
		}
	case token.MUL:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() * yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() * yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) * yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() * uint64(yValue.Int()))
		}
	case token.QUO:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() / yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() / yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) / yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() / uint64(yValue.Int()))
		}
	case token.REM:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() % yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() % yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) % yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() % uint64(yValue.Int()))
		}
	case token.AND:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() & yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() & yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) & yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() & uint64(yValue.Int()))
		}
	case token.OR:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() | yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() | yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) | yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() | uint64(yValue.Int()))
		}
	case token.XOR:
		if xValue.CanInt() && yValue.CanInt() {
			targetValue.SetInt(xValue.Int() ^ yValue.Int())
		} else if xValue.CanUint() && yValue.CanUint() {
			targetValue.SetUint(xValue.Uint() ^ yValue.Uint())
		} else if xValue.CanInt() && yValue.CanUint() {
			targetValue.SetUint(uint64(xValue.Int()) ^ yValue.Uint())
		} else if xValue.CanUint() && yValue.CanInt() {
			targetValue.SetUint(xValue.Uint() ^ uint64(yValue.Int()))
		}
	}
	return targetValue.Interface(), evalType
}
