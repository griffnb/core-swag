package gen

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/console"
	swag "github.com/griffnb/core-swag/internal/legacy_files"
	"github.com/griffnb/core-swag/internal/loader"
	"github.com/griffnb/core-swag/internal/orchestrator"
	"github.com/griffnb/core-swag/internal/schema"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"sigs.k8s.io/yaml"
)

var open = os.Open

// DefaultOverridesFile is the location swagger will look for type overrides.
const DefaultOverridesFile = ".swaggo"

type genTypeWriter func(*Config, *spec.Swagger) error

// Gen presents a generate tool for swag.
type Gen struct {
	json          func(data interface{}) ([]byte, error)
	jsonIndent    func(data interface{}) ([]byte, error)
	jsonToYAML    func(data []byte) ([]byte, error)
	outputTypeMap map[string]genTypeWriter
	debug         Debugger
}

// Debugger is the interface that wraps the basic Printf method.
type Debugger interface {
	Printf(format string, v ...interface{})
}

// New creates a new Gen.
func New() *Gen {
	gen := Gen{
		json: json.Marshal,
		jsonIndent: func(data interface{}) ([]byte, error) {
			return json.MarshalIndent(data, "", "    ")
		},
		jsonToYAML: yaml.JSONToYAML,
		debug:      log.New(os.Stdout, "", log.LstdFlags),
	}

	gen.outputTypeMap = map[string]genTypeWriter{
		"go":   gen.writeDocSwagger,
		"json": gen.writeJSONSwagger,
		"yaml": gen.writeYAMLSwagger,
		"yml":  gen.writeYAMLSwagger,
	}

	return &gen
}

// Config presents Gen configurations.
type Config struct {
	Debugger swag.Debugger

	// SearchDir the swag would parse,comma separated if multiple
	SearchDir string

	// excludes dirs and files in SearchDir,comma separated
	Excludes string

	// outputs only specific extension
	ParseExtension string

	// OutputDir represents the output directory for all the generated files
	OutputDir string

	// OutputTypes define types of files which should be generated
	OutputTypes []string

	// MainAPIFile the Go file path in which 'swagger general API Info' is written
	MainAPIFile string

	// PropNamingStrategy represents property naming strategy like snake case,camel case,pascal case
	PropNamingStrategy string

	// MarkdownFilesDir used to find markdown files, which can be used for tag descriptions
	MarkdownFilesDir string

	// CodeExampleFilesDir used to find code example files, which can be used for x-codeSamples
	CodeExampleFilesDir string

	// InstanceName is used to get distinct names for different swagger documents in the
	// same project. The default value is "swagger".
	InstanceName string

	// ParseDepth dependency parse depth
	ParseDepth int

	// ParseVendor whether swag should be parse vendor folder
	ParseVendor bool

	// ParseDependencies whether swag should be parse outside dependency folder: 0 none, 1 models, 2 operations, 3 all
	ParseDependency int

	// UseStructNames stick to the struct name instead of those ugly full-path names
	UseStructNames bool

	// ParseInternal whether swag should parse internal packages
	ParseInternal bool

	// Strict whether swag should error or warn when it detects cases which are most likely user errors
	Strict bool

	// GeneratedTime whether swag should generate the timestamp at the top of docs.go
	GeneratedTime bool

	// RequiredByDefault set validation required for all fields by default
	RequiredByDefault bool

	// OverridesFile defines global type overrides.
	OverridesFile string

	// ParseGoList whether swag use go list to parse dependency
	ParseGoList bool

	// include only tags mentioned when searching, comma separated
	Tags string

	// LeftTemplateDelim defines the left delimiter for the template generation
	LeftTemplateDelim string

	// RightTemplateDelim defines the right delimiter for the template generation
	RightTemplateDelim string

	// PackageName defines package name of generated `docs.go`
	PackageName string

	// CollectionFormat set default collection format
	CollectionFormat string

	// Parse only packages whose import path match the given prefix, comma separated
	PackagePrefix string

	// State set host state
	State string

	// ParseFuncBody whether swag should parse api info inside of funcs
	ParseFuncBody bool

	// ParseGoPackages whether swag use golang.org/x/tools/go/packages to parse source.
	ParseGoPackages bool
}

// Build builds swagger json file  for given searchDir and mainAPIFile. Returns json.
func (g *Gen) Build(config *Config) error {
	if config.Debugger != nil {
		g.debug = config.Debugger
	}
	if config.InstanceName == "" {
		config.InstanceName = swag.Name
	}

	searchDirs := strings.Split(config.SearchDir, ",")
	if !config.ParseGoPackages { // packages.Load support pattern like ./...
		for _, searchDir := range searchDirs {
			if _, err := os.Stat(searchDir); os.IsNotExist(err) {
				return fmt.Errorf("dir: %s does not exist", searchDir)
			}
		}
	}

	if config.LeftTemplateDelim == "" {
		config.LeftTemplateDelim = "{{"
	}

	if config.RightTemplateDelim == "" {
		config.RightTemplateDelim = "}}"
	}

	var overrides map[string]string

	if config.OverridesFile != "" {
		overridesFile, err := open(config.OverridesFile)
		if err != nil {
			// Don't bother reporting if the default file is missing; assume there are no overrides
			if !(config.OverridesFile == DefaultOverridesFile && os.IsNotExist(err)) {
				return fmt.Errorf("could not open overrides file: %w", err)
			}
		} else {
			console.Logger.Debug("Using overrides from %s", config.OverridesFile)

			overrides, err = parseOverrides(overridesFile)
			if err != nil {
				return err
			}
		}
	}

	console.Logger.Debug("Generate swagger docs....")

	// Create orchestrator with configuration
	orc := orchestrator.New(&orchestrator.Config{
		ParseVendor:             config.ParseVendor,
		ParseInternal:           config.ParseInternal,
		ParseDependency:         loader.ParseFlag(config.ParseDependency),
		PropNamingStrategy:      config.PropNamingStrategy,
		RequiredByDefault:       config.RequiredByDefault,
		Strict:                  config.Strict,
		MarkdownFileDir:         config.MarkdownFilesDir,
		CodeExampleFilesDir:     config.CodeExampleFilesDir,
		CollectionFormatInQuery: config.CollectionFormat,
		Excludes:                parseExcludes(config.Excludes),
		PackagePrefix:           parsePackagePrefix(config.PackagePrefix),
		ParseExtension:          config.ParseExtension,
		ParseGoList:             config.ParseGoList,
		ParseGoPackages:         config.ParseGoPackages,
		HostState:               config.State,
		ParseFuncBody:           config.ParseFuncBody,
		UseStructName:           config.UseStructNames,
		Overrides:               overrides,
		Tags:                    parseTags(config.Tags),
		Debug:                   g.debug,
	})

	// Parse using orchestrator
	swagger, err := orc.Parse(searchDirs, config.MainAPIFile, config.ParseDepth)
	if err != nil {
		return err
	}

	// Sanitize swagger spec to remove infinity/NaN values before any output
	// These values are not valid in JSON and will cause marshaling errors
	g.debug.Printf("Sanitizing swagger spec to remove invalid numeric values...")
	sanitizeSwaggerSpec(swagger)

	// Remove unused schema definitions to keep the output clean
	schema.RemoveUnusedDefinitions(swagger)

	if err := os.MkdirAll(config.OutputDir, os.ModePerm); err != nil {
		return err
	}

	for _, outputType := range config.OutputTypes {
		outputType = strings.ToLower(strings.TrimSpace(outputType))
		if typeWriter, ok := g.outputTypeMap[outputType]; ok {
			if err := typeWriter(config, swagger); err != nil {
				return err
			}
		} else {
			log.Printf("output type '%s' not supported", outputType)
		}
	}

	return nil
}

func (g *Gen) writeDocSwagger(config *Config, swagger *spec.Swagger) error {
	filename := "docs.go"

	if config.State != "" {
		filename = config.State + "_" + filename
	}

	if config.InstanceName != swag.Name {
		filename = config.InstanceName + "_" + filename
	}

	docFileName := path.Join(config.OutputDir, filename)

	absOutputDir, err := filepath.Abs(config.OutputDir)
	if err != nil {
		return err
	}

	var packageName string
	if len(config.PackageName) > 0 {
		packageName = config.PackageName
	} else {
		packageName = filepath.Base(absOutputDir)
		packageName = strings.ReplaceAll(packageName, "-", "_")
	}

	docs, err := os.Create(docFileName)
	if err != nil {
		return err
	}
	defer docs.Close()

	// Write doc
	err = g.writeGoDoc(packageName, docs, swagger, config)
	if err != nil {
		return err
	}

	console.Logger.Debug("create docs.go at %+v", docFileName)

	return nil
}

func (g *Gen) writeJSONSwagger(config *Config, swagger *spec.Swagger) error {
	filename := "swagger.json"

	if config.State != "" {
		filename = config.State + "_" + filename
	}

	if config.InstanceName != swag.Name {
		filename = config.InstanceName + "_" + filename
	}

	jsonFileName := path.Join(config.OutputDir, filename)

	b, err := g.jsonIndent(swagger)
	if err != nil {
		return err
	}

	err = g.writeFile(b, jsonFileName)
	if err != nil {
		return err
	}

	console.Logger.Debug("create swagger.json at %+v", jsonFileName)

	return nil
}

func (g *Gen) writeYAMLSwagger(config *Config, swagger *spec.Swagger) error {
	filename := "swagger.yaml"

	if config.State != "" {
		filename = config.State + "_" + filename
	}

	if config.InstanceName != swag.Name {
		filename = config.InstanceName + "_" + filename
	}

	yamlFileName := path.Join(config.OutputDir, filename)

	b, err := g.json(swagger)
	if err != nil {
		return err
	}

	y, err := g.jsonToYAML(b)
	if err != nil {
		return fmt.Errorf("cannot covert json to yaml error: %s", err)
	}

	err = g.writeFile(y, yamlFileName)
	if err != nil {
		return err
	}

	console.Logger.Debug("create swagger.yaml at %+v", yamlFileName)

	return nil
}

func (g *Gen) writeFile(b []byte, file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write(b)

	return err
}

func (g *Gen) formatSource(src []byte) []byte {
	code, err := format.Source(src)
	if err != nil {
		code = src // Formatter failed, return original code.
	}

	return code
}

// Read and parse the overrides file.
func parseOverrides(r io.Reader) (map[string]string, error) {
	overrides := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		if len(line) > 1 && line[0:2] == "//" {
			continue
		}

		parts := strings.Fields(line)

		switch len(parts) {
		case 0:
			// only whitespace
			continue
		case 2:
			// either a skip or malformed
			if parts[0] != "skip" {
				return nil, fmt.Errorf("could not parse override: '%s'", line)
			}

			overrides[parts[1]] = ""
		case 3:
			// either a replace or malformed
			if parts[0] != "replace" {
				return nil, fmt.Errorf("could not parse override: '%s'", line)
			}

			overrides[parts[1]] = parts[2]
		default:
			return nil, fmt.Errorf("could not parse override: '%s'", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading overrides file: %w", err)
	}

	return overrides, nil
}

func (g *Gen) writeGoDoc(packageName string, output io.Writer, swagger *spec.Swagger, config *Config) error {
	generator, err := template.New("swagger_info").Funcs(template.FuncMap{
		"printDoc": func(v string) string {
			// Add schemes
			v = "{\n    \"schemes\": " + config.LeftTemplateDelim + " marshal .Schemes " + config.RightTemplateDelim + "," + v[1:]
			// Sanitize backticks
			return strings.Replace(v, "`", "`+\"`\"+`", -1)
		},
	}).Parse(packageTemplate)
	if err != nil {
		return err
	}

	swaggerSpec := &spec.Swagger{
		VendorExtensible: swagger.VendorExtensible,
		SwaggerProps: spec.SwaggerProps{
			ID:       swagger.ID,
			Consumes: swagger.Consumes,
			Produces: swagger.Produces,
			Swagger:  swagger.Swagger,
			Info: &spec.Info{
				VendorExtensible: swagger.Info.VendorExtensible,
				InfoProps: spec.InfoProps{
					Description:    config.LeftTemplateDelim + "escape .Description" + config.RightTemplateDelim,
					Title:          config.LeftTemplateDelim + ".Title" + config.RightTemplateDelim,
					TermsOfService: swagger.Info.TermsOfService,
					Contact:        swagger.Info.Contact,
					License:        swagger.Info.License,
					Version:        config.LeftTemplateDelim + ".Version" + config.RightTemplateDelim,
				},
			},
			Host:                config.LeftTemplateDelim + ".Host" + config.RightTemplateDelim,
			BasePath:            config.LeftTemplateDelim + ".BasePath" + config.RightTemplateDelim,
			Paths:               swagger.Paths,
			Definitions:         swagger.Definitions,
			Parameters:          swagger.Parameters,
			Responses:           swagger.Responses,
			SecurityDefinitions: swagger.SecurityDefinitions,
			Security:            swagger.Security,
			Tags:                swagger.Tags,
			ExternalDocs:        swagger.ExternalDocs,
		},
	}

	// crafted docs.json
	buf, err := g.jsonIndent(swaggerSpec)
	if err != nil {
		return err
	}

	state := ""
	if len(config.State) > 0 {
		state = cases.Title(language.English).String(strings.ToLower(config.State))
	}

	buffer := &bytes.Buffer{}

	err = generator.Execute(buffer, struct {
		Timestamp          time.Time
		Doc                string
		Host               string
		PackageName        string
		BasePath           string
		Title              string
		Description        string
		Version            string
		State              string
		InstanceName       string
		Schemes            []string
		GeneratedTime      bool
		LeftTemplateDelim  string
		RightTemplateDelim string
	}{
		Timestamp:          time.Now(),
		GeneratedTime:      config.GeneratedTime,
		Doc:                string(buf),
		Host:               swagger.Host,
		PackageName:        packageName,
		BasePath:           swagger.BasePath,
		Schemes:            swagger.Schemes,
		Title:              swagger.Info.Title,
		Description:        swagger.Info.Description,
		Version:            swagger.Info.Version,
		State:              state,
		InstanceName:       config.InstanceName,
		LeftTemplateDelim:  config.LeftTemplateDelim,
		RightTemplateDelim: config.RightTemplateDelim,
	})
	if err != nil {
		return err
	}

	code := g.formatSource(buffer.Bytes())

	// write
	_, err = output.Write(code)

	return err
}

var packageTemplate = `// Package {{.PackageName}} Code generated by swaggo/swag{{ if .GeneratedTime }} at {{ .Timestamp }}{{ end }}. DO NOT EDIT
package {{.PackageName}}

import "github.com/griffnb/core-swag"

const docTemplate{{ if ne .InstanceName "swagger" }}{{ .InstanceName }} {{- end }}{{ .State }} = ` + "`{{ printDoc .Doc}}`" + `

// Swagger{{ .State }}Info{{ if ne .InstanceName "swagger" }}{{ .InstanceName }} {{- end }} holds exported Swagger Info so clients can modify it
var Swagger{{ .State }}Info{{ if ne .InstanceName "swagger" }}{{ .InstanceName }} {{- end }} = &swag.Spec{
	Version:     {{ printf "%q" .Version}},
	Host:        {{ printf "%q" .Host}},
	BasePath:    {{ printf "%q" .BasePath}},
	Schemes:     []string{ {{ range $index, $schema := .Schemes}}{{if gt $index 0}},{{end}}{{printf "%q" $schema}}{{end}} },
	Title:       {{ printf "%q" .Title}},
	Description: {{ printf "%q" .Description}},
	InfoInstanceName: {{ printf "%q" .InstanceName }},
	SwaggerTemplate: docTemplate{{ if ne .InstanceName "swagger" }}{{ .InstanceName }} {{- end }}{{ .State }},
	LeftDelim:        {{ printf "%q" .LeftTemplateDelim}},
	RightDelim:       {{ printf "%q" .RightTemplateDelim}},
}

func init() {
	swag.Register(Swagger{{ .State }}Info{{ if ne .InstanceName "swagger" }}{{ .InstanceName }} {{- end }}.InstanceName(), Swagger{{ .State }}Info{{ if ne .InstanceName "swagger" }}{{ .InstanceName }} {{- end }})
}
`

// parseExcludes converts comma-separated exclude string to map.
func parseExcludes(excludes string) map[string]struct{} {
	result := make(map[string]struct{})
	if excludes == "" {
		return result
	}

	for _, exclude := range strings.Split(excludes, ",") {
		exclude = strings.TrimSpace(exclude)
		if exclude != "" {
			result[exclude] = struct{}{}
		}
	}
	return result
}

// parsePackagePrefix converts comma-separated prefix string to slice.
func parsePackagePrefix(packagePrefix string) []string {
	if packagePrefix == "" {
		return []string{}
	}

	result := []string{}
	for _, prefix := range strings.Split(packagePrefix, ",") {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" {
			result = append(result, prefix)
		}
	}
	return result
}

// parseTags converts comma-separated tags string to map.
func parseTags(tags string) map[string]struct{} {
	result := make(map[string]struct{})
	if tags == "" {
		return result
	}

	for _, tag := range strings.Split(tags, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result[tag] = struct{}{}
		}
	}
	return result
}

// sanitizeSwaggerSpec removes infinity and NaN values from the swagger spec
// to prevent JSON marshaling errors. These values are not valid in JSON.
func sanitizeSwaggerSpec(swagger *spec.Swagger) {
	if swagger == nil || swagger.Paths == nil {
		return
	}

	// Walk through all paths and operations
	for pathKey, pathItem := range swagger.Paths.Paths {
		sanitizeOperation(pathKey, pathItem.Get)
		sanitizeOperation(pathKey, pathItem.Put)
		sanitizeOperation(pathKey, pathItem.Post)
		sanitizeOperation(pathKey, pathItem.Delete)
		sanitizeOperation(pathKey, pathItem.Options)
		sanitizeOperation(pathKey, pathItem.Head)
		sanitizeOperation(pathKey, pathItem.Patch)
	}
}

// sanitizeOperation sanitizes parameters in an operation
func sanitizeOperation(path string, op *spec.Operation) {
	if op == nil {
		return
	}

	for i := range op.Parameters {
		param := &op.Parameters[i]
		sanitizeParameter(path, param)
	}
}

// sanitizeParameter removes infinity/NaN values from parameter constraints
func sanitizeParameter(path string, param *spec.Parameter) {
	if param == nil {
		return
	}

	// Check and clear invalid Minimum values
	if param.Minimum != nil && (math.IsInf(*param.Minimum, 0) || math.IsNaN(*param.Minimum)) {
		param.Minimum = nil
	}

	// Check and clear invalid Maximum values
	if param.Maximum != nil && (math.IsInf(*param.Maximum, 0) || math.IsNaN(*param.Maximum)) {
		param.Maximum = nil
	}

	// Check and clear invalid MultipleOf values
	if param.MultipleOf != nil && (math.IsInf(*param.MultipleOf, 0) || math.IsNaN(*param.MultipleOf)) {
		param.MultipleOf = nil
	}

	// Check and clear invalid Default values if they're numeric
	if param.Default != nil {
		if f, ok := param.Default.(float64); ok && (math.IsInf(f, 0) || math.IsNaN(f)) {
			param.Default = nil
		}
	}

	// Check and clear invalid Example values if they're numeric
	if param.Example != nil {
		if f, ok := param.Example.(float64); ok && (math.IsInf(f, 0) || math.IsNaN(f)) {
			param.Example = nil
		}
	}

	// Check and sanitize Enum values
	if len(param.Enum) > 0 {
		validEnum := make([]interface{}, 0, len(param.Enum))
		for _, enumVal := range param.Enum {
			if f, ok := enumVal.(float64); ok && (math.IsInf(f, 0) || math.IsNaN(f)) {
				continue // Skip infinity/NaN enum values
			}
			validEnum = append(validEnum, enumVal)
		}
		param.Enum = validEnum
	}

	// Sanitize schema if present (for body parameters)
	if param.Schema != nil {
		sanitizeSchema(param.Schema)
	}

	// Sanitize items if present (for array parameters)
	if param.Items != nil {
		sanitizeItems(param.Items)
	}
}

// sanitizeSchema recursively sanitizes a schema
func sanitizeSchema(schema *spec.Schema) {
	if schema == nil {
		return
	}

	// Sanitize numeric constraints
	if schema.Minimum != nil && (math.IsInf(*schema.Minimum, 0) || math.IsNaN(*schema.Minimum)) {
		schema.Minimum = nil
	}
	if schema.Maximum != nil && (math.IsInf(*schema.Maximum, 0) || math.IsNaN(*schema.Maximum)) {
		schema.Maximum = nil
	}
	if schema.MultipleOf != nil && (math.IsInf(*schema.MultipleOf, 0) || math.IsNaN(*schema.MultipleOf)) {
		schema.MultipleOf = nil
	}

	// Sanitize default if numeric
	if schema.Default != nil {
		if f, ok := schema.Default.(float64); ok && (math.IsInf(f, 0) || math.IsNaN(f)) {
			schema.Default = nil
		}
	}

	// Recursively sanitize properties
	if schema.Properties != nil {
		for k := range schema.Properties {
			propSchema := schema.Properties[k]
			sanitizeSchema(&propSchema)
			schema.Properties[k] = propSchema
		}
	}

	// Sanitize array items
	if schema.Items != nil && schema.Items.Schema != nil {
		sanitizeSchema(schema.Items.Schema)
	}

	// Sanitize AllOf schemas
	for i := range schema.AllOf {
		sanitizeSchema(&schema.AllOf[i])
	}

	// Sanitize AnyOf schemas
	for i := range schema.AnyOf {
		sanitizeSchema(&schema.AnyOf[i])
	}

	// Sanitize OneOf schemas
	for i := range schema.OneOf {
		sanitizeSchema(&schema.OneOf[i])
	}

	// Sanitize Not schema
	if schema.Not != nil {
		sanitizeSchema(schema.Not)
	}

	// Sanitize AdditionalProperties if it's a schema
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		sanitizeSchema(schema.AdditionalProperties.Schema)
	}
}

// sanitizeItems sanitizes items in array parameters
func sanitizeItems(items *spec.Items) {
	if items == nil {
		return
	}

	if items.Minimum != nil && (math.IsInf(*items.Minimum, 0) || math.IsNaN(*items.Minimum)) {
		items.Minimum = nil
	}
	if items.Maximum != nil && (math.IsInf(*items.Maximum, 0) || math.IsNaN(*items.Maximum)) {
		items.Maximum = nil
	}
	if items.MultipleOf != nil && (math.IsInf(*items.MultipleOf, 0) || math.IsNaN(*items.MultipleOf)) {
		items.MultipleOf = nil
	}

	if items.Default != nil {
		if f, ok := items.Default.(float64); ok && (math.IsInf(f, 0) || math.IsNaN(f)) {
			items.Default = nil
		}
	}

	if items.Example != nil {
		if f, ok := items.Example.(float64); ok && (math.IsInf(f, 0) || math.IsNaN(f)) {
			items.Example = nil
		}
	}

	// Recursively sanitize nested items
	if items.Items != nil {
		sanitizeItems(items.Items)
	}
}
