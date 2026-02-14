package base

import (
	"fmt"
	"strings"

	"github.com/go-openapi/spec"
)

func (s *Service) parseSecurityDefinition(context string, lines []string, index *int) (*spec.SecurityScheme, error) {
	const (
		in               = "@in"
		name             = "@name"
		descriptionAttr  = "@description"
		tokenURL         = "@tokenurl"
		authorizationURL = "@authorizationurl"
	)

	var search []string

	attribute := strings.ToLower(FieldsByAnySpace(lines[*index], 2)[0])
	switch attribute {
	case "@securitydefinitions.basic":
		return spec.BasicAuth(), nil
	case "@securitydefinitions.apikey":
		search = []string{in, name}
	case "@securitydefinitions.oauth2.application", "@securitydefinitions.oauth2.password":
		search = []string{tokenURL}
	case "@securitydefinitions.oauth2.implicit":
		search = []string{authorizationURL}
	case "@securitydefinitions.oauth2.accesscode":
		search = []string{tokenURL, authorizationURL}
	}

	// For the first line we get the attributes in the context parameter, so we skip to the next one
	*index++

	attrMap, scopes := make(map[string]string), make(map[string]string)
	extensions, description := make(map[string]interface{}), ""

loopline:
	for ; *index < len(lines); *index++ {
		v := strings.TrimSpace(lines[*index])
		if len(v) == 0 {
			continue
		}

		fields := FieldsByAnySpace(v, 2)
		securityAttr := strings.ToLower(fields[0])
		var value string
		if len(fields) > 1 {
			value = fields[1]
		}

		for _, findterm := range search {
			if securityAttr == findterm {
				attrMap[securityAttr] = value
				continue loopline
			}
		}

		if isExists, err := isExistsScope(securityAttr); err != nil {
			return nil, err
		} else if isExists {
			scopes[securityAttr[len("@scope."):]] = value
			continue
		}

		if strings.HasPrefix(securityAttr, "@x-") {
			extensions[securityAttr[1:]] = value
			continue
		}

		if securityAttr == descriptionAttr {
			if description != "" {
				description += "\n"
			}
			description += value
		}

		if strings.Index(securityAttr, "@securitydefinitions.") == 0 {
			*index--
			break
		}
	}

	if len(attrMap) != len(search) {
		return nil, fmt.Errorf("%s is %v required", context, search)
	}

	var scheme *spec.SecurityScheme

	switch attribute {
	case "@securitydefinitions.apikey":
		scheme = spec.APIKeyAuth(attrMap[name], attrMap[in])
	case "@securitydefinitions.oauth2.application":
		scheme = spec.OAuth2Application(attrMap[tokenURL])
	case "@securitydefinitions.oauth2.implicit":
		scheme = spec.OAuth2Implicit(attrMap[authorizationURL])
	case "@securitydefinitions.oauth2.password":
		scheme = spec.OAuth2Password(attrMap[tokenURL])
	case "@securitydefinitions.oauth2.accesscode":
		scheme = spec.OAuth2AccessToken(attrMap[authorizationURL], attrMap[tokenURL])
	}

	scheme.Description = description

	for extKey, extValue := range extensions {
		scheme.AddExtension(extKey, extValue)
	}

	for scope, scopeDescription := range scopes {
		scheme.AddScope(scope, scopeDescription)
	}

	return scheme, nil
}

func isExistsScope(scope string) (bool, error) {
	s := strings.Fields(scope)
	for _, v := range s {
		if strings.HasPrefix(v, "@scope.") {
			if strings.Contains(v, ",") {
				return false, fmt.Errorf("@scope can't use comma(,) get=%s", v)
			}
		}
	}

	return strings.HasPrefix(scope, "@scope."), nil
}
