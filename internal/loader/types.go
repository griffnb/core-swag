package loader

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// ParseFlag determines what to parse
type ParseFlag int

const (
	// ParseNone parse nothing
	ParseNone ParseFlag = 0x00
	// ParseModels parse models
	ParseModels = 0x01
	// ParseOperations parse operations
	ParseOperations = 0x02
	// ParseAll parse operations and models
	ParseAll = ParseOperations | ParseModels
)

// Service handles loading Go packages and their AST files
type Service struct {
	parseVendor     bool
	parseInternal   bool
	excludes        map[string]struct{}
	packagePrefix   []string
	parseExtension  string
	useGoList       bool
	useGoPackages   bool
	parseDependency ParseFlag
	debug           Debugger
}

// Debugger interface for logging
type Debugger interface {
	Printf(format string, v ...interface{})
}

// LoadResult contains the results of loading packages
type LoadResult struct {
	Files    map[*ast.File]*AstFileInfo
	Packages []*packages.Package
}

// AstFileInfo contains information about a parsed AST file
type AstFileInfo struct {
	File        *ast.File
	Path        string
	PackagePath string
	ParseFlag   ParseFlag
	FileSet     *token.FileSet
}

// Option is a functional option for configuring Service
type Option func(*Service)

// noOpDebugger is a no-op debugger
type noOpDebugger struct{}

func (n *noOpDebugger) Printf(format string, v ...interface{}) {}
