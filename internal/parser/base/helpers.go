package base

import (
	"fmt"
	"strings"
)

// isGeneralAPIComment checks if comments contain general API info
func isGeneralAPIComment(comments []string) bool {
	for _, commentLine := range comments {
		commentLine = strings.TrimSpace(commentLine)
		if len(commentLine) == 0 {
			continue
		}
		attribute := strings.ToLower(FieldsByAnySpace(commentLine, 2)[0])
		switch attribute {
		case "@summary", "@router", "@success", "@failure", "@response":
			return false
		}
	}
	return true
}

// parseMimeTypeList parses comma-separated MIME types and their aliases
func parseMimeTypeList(commentLine string, mimeTypes *[]string) error {
	for _, typeName := range strings.Split(commentLine, ",") {
		typeName = strings.TrimSpace(typeName)
		if typeName == "" {
			continue
		}
		if mimeTypePattern.MatchString(typeName) {
			*mimeTypes = append(*mimeTypes, typeName)
			continue
		}

		aliasMimeType, ok := mimeTypeAliases[typeName]
		if !ok {
			return fmt.Errorf("%v accept type can't be accepted", typeName)
		}
		*mimeTypes = append(*mimeTypes, aliasMimeType)
	}
	return nil
}

// parseSecurity parses security requirements from comment line
func parseSecurity(commentLine string) map[string][]string {
	securityMap := make(map[string][]string)

	for _, securityOption := range securityPairSepPattern.Split(commentLine, -1) {
		securityOption = strings.TrimSpace(securityOption)

		left, right := strings.Index(securityOption, "["), strings.Index(securityOption, "]")

		if !(left == -1 && right == -1) {
			scopes := securityOption[left+1 : right]
			var options []string
			for _, scope := range strings.Split(scopes, ",") {
				options = append(options, strings.TrimSpace(scope))
			}
			securityKey := securityOption[0:left]
			securityMap[securityKey] = append(securityMap[securityKey], options...)
		} else {
			securityKey := strings.TrimSpace(securityOption)
			securityMap[securityKey] = []string{}
		}
	}

	return securityMap
}
