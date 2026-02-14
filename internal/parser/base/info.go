package base

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-openapi/spec"
)

// setSwaggerInfo sets various swagger info fields based on the attribute
func (s *Service) setSwaggerInfo(attribute, value string) {
	switch attribute {
	case "@version":
		s.swagger.Info.Version = value
	case "@title":
		s.swagger.Info.Title = value
	case "@termsofservice":
		s.swagger.Info.TermsOfService = value
	case "@description":
		s.swagger.Info.Description = value
	case "@contact.name":
		s.swagger.Info.Contact.Name = value
	case "@contact.email":
		s.swagger.Info.Contact.Email = value
	case "@contact.url":
		s.swagger.Info.Contact.URL = value
	case "@license.name":
		s.swagger.Info.License = initIfEmpty(s.swagger.Info.License)
		s.swagger.Info.License.Name = value
	case "@license.url":
		s.swagger.Info.License = initIfEmpty(s.swagger.Info.License)
		s.swagger.Info.License.URL = value
	}
}

// getMarkdownForTag reads markdown content for a given tag name
func (s *Service) getMarkdownForTag(tagName string) ([]byte, error) {
	if tagName == "" {
		return make([]byte, 0), nil
	}

	dirEntries, err := os.ReadDir(s.markdownFileDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		expectedFileName := tagName
		if !strings.HasSuffix(tagName, ".md") {
			expectedFileName = tagName + ".md"
		}

		if fileName == expectedFileName {
			fullPath := filepath.Join(s.markdownFileDir, fileName)
			commentInfo, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, fmt.Errorf("Failed to read markdown file %s error: %s ", fullPath, err)
			}
			return commentInfo, nil
		}
	}

	return nil, fmt.Errorf("Unable to find markdown file for tag %s in the given directory", tagName)
}

// initIfEmpty initializes a license if it's nil
func initIfEmpty(license *spec.License) *spec.License {
	if license == nil {
		return new(spec.License)
	}
	return license
}
