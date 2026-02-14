// Package registry - external package dependency loading.
package registry

import (
	goparser "go/parser"
	"os"
	"strings"

	"golang.org/x/tools/go/loader"
)

func (s *Service) loadExternalPackage(importPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	conf := loader.Config{
		ParserMode: goparser.ParseComments,
		Cwd:        cwd,
	}

	conf.Import(importPath)

	loaderProgram, err := conf.Load()
	if err != nil {
		return err
	}

	for _, info := range loaderProgram.AllPackages {
		pkgPath := strings.TrimPrefix(info.Pkg.Path(), "vendor/")
		for _, astFile := range info.Files {
			s.parseTypesFromFile(astFile, pkgPath, nil)
		}
	}

	return nil
}
