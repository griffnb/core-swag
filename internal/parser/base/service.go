package base

import (
	"fmt"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/go-openapi/spec"
)

var (
	securityPairSepPattern = regexp.MustCompile(`\|\||&&`)
	mimeTypePattern        = regexp.MustCompile("^[^/]+/[^/]+$")
)

var mimeTypeAliases = map[string]string{
	"json":                  "application/json",
	"xml":                   "text/xml",
	"plain":                 "text/plain",
	"html":                  "text/html",
	"mpfd":                  "multipart/form-data",
	"x-www-form-urlencoded": "application/x-www-form-urlencoded",
	"json-api":              "application/vnd.api+json",
	"json-stream":           "application/x-json-stream",
	"octet-stream":          "application/octet-stream",
	"png":                   "image/png",
	"jpeg":                  "image/jpeg",
	"gif":                   "image/gif",
	"event-stream":          "text/event-stream",
}

// Debugger interface for logging
type Debugger interface {
	Printf(format string, v ...interface{})
}

// Service handles parsing of general API information from comments
type Service struct {
	swagger         *spec.Swagger
	markdownFileDir string
	debug           Debugger
}

// NewService creates a new base parser service
func NewService(swagger *spec.Swagger) *Service {
	return &Service{
		swagger: swagger,
	}
}

// SetMarkdownFileDir sets the directory for markdown files
func (s *Service) SetMarkdownFileDir(dir string) {
	s.markdownFileDir = dir
}

// SetDebugger sets the debugger for logging
func (s *Service) SetDebugger(debug Debugger) {
	s.debug = debug
}

// ParseGeneralInfo parses general API info from comment lines
func (s *Service) ParseGeneralInfo(comments []string) error {
	previousAttribute := ""
	var tag *spec.Tag

	for line := 0; line < len(comments); line++ {
		commentLine := comments[line]
		commentLine = strings.TrimSpace(commentLine)
		if len(commentLine) == 0 {
			continue
		}
		fields := FieldsByAnySpace(commentLine, 2)

		attribute := fields[0]
		var value string
		if len(fields) > 1 {
			value = fields[1]
		}

		switch attr := strings.ToLower(attribute); attr {
		case "@version", "@title", "@termsofservice", "@license.name", "@license.url",
			"@contact.name", "@contact.url", "@contact.email":
			s.setSwaggerInfo(attr, value)

		case "@description":
			if previousAttribute == attribute {
				s.swagger.Info.Description = AppendDescription(s.swagger.Info.Description, value)
				continue
			}
			s.setSwaggerInfo(attr, value)

		case "@description.markdown":
			commentInfo, err := s.getMarkdownForTag("api")
			if err != nil {
				return err
			}
			s.setSwaggerInfo("@description", string(commentInfo))

		case "@host":
			s.swagger.Host = value

		case "@basepath":
			s.swagger.BasePath = value

		case "@accept":
			return parseMimeTypeList(value, &s.swagger.Consumes)

		case "@produce":
			return parseMimeTypeList(value, &s.swagger.Produces)

		case "@schemes":
			s.swagger.Schemes = strings.Split(value, " ")

		case "@tag.name":
			s.swagger.Tags = append(s.swagger.Tags, spec.Tag{
				TagProps: spec.TagProps{
					Name: value,
				},
			})
			tag = &s.swagger.Tags[len(s.swagger.Tags)-1]

		case "@tag.description":
			if tag != nil {
				tag.TagProps.Description = value
			}

		case "@tag.description.markdown":
			if tag != nil {
				commentInfo, err := s.getMarkdownForTag(tag.TagProps.Name)
				if err != nil {
					return err
				}
				tag.TagProps.Description = string(commentInfo)
			}

		case "@tag.docs.url":
			if tag != nil {
				tag.TagProps.ExternalDocs = &spec.ExternalDocumentation{
					URL: value,
				}
			}

		case "@tag.docs.description":
			if tag != nil {
				if tag.TagProps.ExternalDocs == nil {
					return fmt.Errorf("%s needs to come after a @tags.docs.url", attribute)
				}
				tag.TagProps.ExternalDocs.Description = value
			}

		case "@securitydefinitions.basic", "@securitydefinitions.apikey",
			"@securitydefinitions.oauth2.application", "@securitydefinitions.oauth2.implicit",
			"@securitydefinitions.oauth2.password", "@securitydefinitions.oauth2.accesscode":
			scheme, err := s.parseSecurityDefinition(attribute, comments, &line)
			if err != nil {
				return err
			}
			s.swagger.SecurityDefinitions[value] = scheme

		case "@security":
			s.swagger.Security = append(s.swagger.Security, parseSecurity(value))

		case "@externaldocs.description", "@externaldocs.url":
			if s.swagger.ExternalDocs == nil {
				s.swagger.ExternalDocs = new(spec.ExternalDocumentation)
			}
			switch attr {
			case "@externaldocs.description":
				s.swagger.ExternalDocs.Description = value
			case "@externaldocs.url":
				s.swagger.ExternalDocs.URL = value
			}

		default:
			if strings.HasPrefix(attribute, "@x-") {
				if err := s.parseExtension(attribute, value, tag); err != nil {
					return err
				}
			} else if strings.HasPrefix(attribute, "@tag.x-") {
				if err := s.parseTagExtension(attribute, value, tag); err != nil {
					return err
				}
			}
		}

		previousAttribute = attribute
	}

	return nil
}

// ParseGeneralAPIInfo parses general api info for given mainAPIFile path
func (s *Service) ParseGeneralAPIInfo(mainAPIFile string) error {
	fileTree, err := parser.ParseFile(token.NewFileSet(), mainAPIFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("cannot parse source files %s: %s", mainAPIFile, err)
	}

	s.swagger.Swagger = "2.0"

	for _, comment := range fileTree.Comments {
		comments := strings.Split(comment.Text(), "\n")
		if !isGeneralAPIComment(comments) {
			continue
		}

		err = s.ParseGeneralInfo(comments)
		if err != nil {
			return err
		}
	}

	return nil
}
