package base

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-openapi/spec"
)

// parseExtension parses custom @x- extensions
func (s *Service) parseExtension(attribute, value string, tag *spec.Tag) error {
	extensionName := attribute[1:]

	// Check if extension exists in security definitions
	extExistsInSecurityDef := false
	for _, v := range s.swagger.SecurityDefinitions {
		_, extExistsInSecurityDef = v.VendorExtensible.Extensions.GetString(extensionName)
		if extExistsInSecurityDef {
			break
		}
	}

	if extExistsInSecurityDef {
		return nil
	}

	if len(value) == 0 {
		return fmt.Errorf("annotation %s need a value", attribute)
	}

	var valueJSON interface{}
	err := json.Unmarshal([]byte(value), &valueJSON)
	if err != nil {
		return fmt.Errorf("annotation %s need a valid json value", attribute)
	}

	if strings.Contains(extensionName, "logo") {
		s.swagger.Info.Extensions.Add(extensionName, valueJSON)
	} else {
		if s.swagger.Extensions == nil {
			s.swagger.Extensions = make(map[string]interface{})
		}
		s.swagger.Extensions[attribute[1:]] = valueJSON
	}

	return nil
}

// parseTagExtension parses @tag.x- extensions for tags
func (s *Service) parseTagExtension(attribute, value string, tag *spec.Tag) error {
	if tag == nil {
		return nil
	}

	extensionName := attribute[5:]

	if len(value) == 0 {
		return fmt.Errorf("annotation %s need a value", attribute)
	}

	if tag.Extensions == nil {
		tag.Extensions = make(map[string]interface{})
	}

	tag.Extensions[extensionName] = value
	return nil
}
