package loader

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
)

// parseGoFile parses a Go source file
func parseGoFile(fileSet *token.FileSet, path string, src interface{}) (*ast.File, error) {
	return goparser.ParseFile(fileSet, path, src, goparser.ParseComments)
}
