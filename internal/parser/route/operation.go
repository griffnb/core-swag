package route

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/swaggo/swag/internal/parser/route/domain"
)

// operation represents a parsed operation (before being split into routes)
type operation struct {
	functionName string
	packageName  string // Package name for type resolution
	summary      string
	description  string
	tags         []string
	operationID  string
	routerPaths  []routerPath
	parameters   []domain.Parameter
	responses    map[int]domain.Response
	security     []map[string][]string
	consumes     []string
	produces     []string
	isPublic     bool
}

// routerPath represents a single @router annotation
type routerPath struct {
	path       string
	method     string
	deprecated bool
}

var (
	routerPattern = regexp.MustCompile(`^(/[\w./\-{}\(\)+:$~]*)[[:blank:]]+\[(\w+)]`)
	mimeTypeAliases = map[string]string{
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
)

// parseComment parses a single comment line and updates the operation
func (s *Service) parseComment(op *operation, comment string) error {
	// Comments from AST come as "// text" or "/* text */"
	commentLine := strings.TrimSpace(comment)

	// Remove comment markers
	if strings.HasPrefix(commentLine, "//") {
		commentLine = strings.TrimSpace(commentLine[2:])
	} else if strings.HasPrefix(commentLine, "/*") {
		commentLine = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(commentLine, "/*"), "*/"))
	}

	if len(commentLine) == 0 {
		return nil
	}

	// Split into fields
	allFields := strings.Fields(commentLine)
	if len(allFields) == 0 {
		return nil
	}

	attribute := strings.ToLower(allFields[0])
	var lineRemainder string
	if len(allFields) > 1 {
		lineRemainder = strings.Join(allFields[1:], " ")
	}

	switch attribute {
	case "@public":
		op.isPublic = true
	case "@summary":
		op.summary = lineRemainder
	case "@description":
		if op.description == "" {
			op.description = lineRemainder
		} else {
			op.description += "\n" + lineRemainder
		}
	case "@description.markdown":
		// Load description from markdown file
		content, err := s.loadMarkdownFile(lineRemainder)
		if err != nil {
			return err
		}
		op.description = string(content)
	case "@id":
		op.operationID = lineRemainder
	case "@tags":
		op.tags = parseTags(lineRemainder)
	case "@accept":
		return s.parseAccept(op, lineRemainder)
	case "@produce":
		return s.produceParse(op, lineRemainder)
	case "@param":
		return s.parseParam(op, lineRemainder)
	case "@success", "@failure", "@response":
		return s.parseResponse(op, lineRemainder)
	case "@header":
		return s.parseHeader(op, lineRemainder)
	case "@router":
		return s.parseRouter(op, lineRemainder, false)
	case "@deprecatedrouter":
		return s.parseRouter(op, lineRemainder, true)
	case "@security":
		return s.parseSecurity(op, lineRemainder)
	case "@deprecated":
		// Mark all router paths as deprecated
		for i := range op.routerPaths {
			op.routerPaths[i].deprecated = true
		}
	}

	return nil
}

// fieldsByAnySpace splits a string by any whitespace into at most n fields
func fieldsByAnySpace(s string, n int) []string {
	return strings.Fields(strings.TrimSpace(s))[:min(n, len(strings.Fields(s)))]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseTags parses a comma-separated list of tags
func parseTags(line string) []string {
	var tags []string
	for _, tag := range strings.Split(line, ",") {
		tags = append(tags, strings.TrimSpace(tag))
	}
	return tags
}

// parseAccept parses the @accept annotation
func (s *Service) parseAccept(op *operation, line string) error {
	return s.parseMimeTypes(line, &op.consumes)
}

// produceParse parses the @produce annotation
func (s *Service) produceParse(op *operation, line string) error {
	return s.parseMimeTypes(line, &op.produces)
}

// parseMimeTypes parses a comma-separated list of mime types
func (s *Service) parseMimeTypes(line string, target *[]string) error {
	for _, mimeType := range strings.Split(line, ",") {
		mimeType = strings.TrimSpace(mimeType)
		if mimeType == "" {
			continue
		}

		// Check if it's an alias
		if fullType, ok := mimeTypeAliases[mimeType]; ok {
			*target = append(*target, fullType)
		} else {
			*target = append(*target, mimeType)
		}
	}
	return nil
}

// parseRouter parses the @router or @deprecatedrouter annotation
func (s *Service) parseRouter(op *operation, line string, deprecated bool) error {
	matches := routerPattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return fmt.Errorf("can not parse router comment \"%s\"", line)
	}

	path := matches[1]
	method := strings.ToUpper(matches[2])

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}

	op.routerPaths = append(op.routerPaths, routerPath{
		path:       path,
		method:     method,
		deprecated: deprecated,
	})

	return nil
}

// loadMarkdownFile loads content from a markdown file
func (s *Service) loadMarkdownFile(tag string) ([]byte, error) {
	if s.markdownFileDir == "" {
		return nil, fmt.Errorf("markdown file directory not set")
	}

	// The tag is the filename (e.g., "api" becomes "api.md")
	filename := tag + ".md"
	filepath := filepath.Join(s.markdownFileDir, filename)

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown file %s: %w", filepath, err)
	}

	return content, nil
}

// parseSecurity parses the @security annotation
func (s *Service) parseSecurity(op *operation, line string) error {
	if len(line) == 0 {
		op.security = []map[string][]string{}
		return nil
	}

	securityMap := make(map[string][]string)

	// Handle security with scopes: OAuth2[read, write]
	if idx := strings.Index(line, "["); idx != -1 {
		endIdx := strings.Index(line, "]")
		if endIdx == -1 {
			return fmt.Errorf("invalid security annotation: %s", line)
		}

		securityKey := strings.TrimSpace(line[:idx])
		scopesStr := line[idx+1 : endIdx]

		var scopes []string
		for _, scope := range strings.Split(scopesStr, ",") {
			scopes = append(scopes, strings.TrimSpace(scope))
		}

		securityMap[securityKey] = scopes
	} else {
		// Simple security without scopes
		securityKey := strings.TrimSpace(line)
		securityMap[securityKey] = []string{}
	}

	op.security = append(op.security, securityMap)
	return nil
}
