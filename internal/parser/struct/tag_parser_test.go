package structparser

import (
	"reflect"
	"testing"
)

// TestParseJSONTag tests parsing JSON struct tags
func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		name          string
		tag           string
		wantName      string
		wantOmitEmpty bool
		wantIgnore    bool
	}{
		{
			name:          "Simple JSON tag",
			tag:           `json:"first_name"`,
			wantName:      "first_name",
			wantOmitEmpty: false,
			wantIgnore:    false,
		},
		{
			name:          "JSON tag with omitempty",
			tag:           `json:"count,omitempty"`,
			wantName:      "count",
			wantOmitEmpty: true,
			wantIgnore:    false,
		},
		{
			name:          "JSON tag with ignore",
			tag:           `json:"-"`,
			wantName:      "",
			wantOmitEmpty: false,
			wantIgnore:    true,
		},
		{
			name:          "JSON tag with spaces",
			tag:           `json:" name "`,
			wantName:      "name",
			wantOmitEmpty: false,
			wantIgnore:    false,
		},
		{
			name:          "JSON tag with omitempty and spaces",
			tag:           `json:"field_name , omitempty"`,
			wantName:      "field_name",
			wantOmitEmpty: true,
			wantIgnore:    false,
		},
		{
			name:          "JSON tag with multiple options",
			tag:           `json:"value,omitempty,string"`,
			wantName:      "value",
			wantOmitEmpty: true,
			wantIgnore:    false,
		},
		{
			name:          "No JSON tag",
			tag:           `form:"username"`,
			wantName:      "",
			wantOmitEmpty: false,
			wantIgnore:    false,
		},
		{
			name:          "Empty JSON tag",
			tag:           `json:""`,
			wantName:      "",
			wantOmitEmpty: false,
			wantIgnore:    false,
		},
		{
			name:          "JSON tag with just omitempty",
			tag:           `json:",omitempty"`,
			wantName:      "",
			wantOmitEmpty: true,
			wantIgnore:    false,
		},
		{
			name:          "Empty string (no tags)",
			tag:           ``,
			wantName:      "",
			wantOmitEmpty: false,
			wantIgnore:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structTag := reflect.StructTag(tt.tag)
			gotName, gotOmitEmpty, gotIgnore := parseJSONTag(structTag)

			if gotName != tt.wantName {
				t.Errorf("parseJSONTag() name = %v, want %v", gotName, tt.wantName)
			}
			if gotOmitEmpty != tt.wantOmitEmpty {
				t.Errorf("parseJSONTag() omitEmpty = %v, want %v", gotOmitEmpty, tt.wantOmitEmpty)
			}
			if gotIgnore != tt.wantIgnore {
				t.Errorf("parseJSONTag() ignore = %v, want %v", gotIgnore, tt.wantIgnore)
			}
		})
	}
}

// TestParsePublicTag tests parsing public struct tags
func TestParsePublicTag(t *testing.T) {
	tests := []struct {
		name           string
		tag            string
		wantVisibility string
	}{
		{
			name:           "Public view tag",
			tag:            `public:"view"`,
			wantVisibility: "view",
		},
		{
			name:           "Public edit tag",
			tag:            `public:"edit"`,
			wantVisibility: "edit",
		},
		{
			name:           "No public tag (private by default)",
			tag:            `json:"name"`,
			wantVisibility: "private",
		},
		{
			name:           "Empty public tag",
			tag:            `public:""`,
			wantVisibility: "private",
		},
		{
			name:           "Public tag with spaces",
			tag:            `public:" view "`,
			wantVisibility: "view",
		},
		{
			name:           "Multiple tags including public",
			tag:            `json:"name" public:"edit" validate:"required"`,
			wantVisibility: "edit",
		},
		{
			name:           "Empty string (no tags)",
			tag:            ``,
			wantVisibility: "private",
		},
		{
			name:           "Public tag with uppercase (should normalize)",
			tag:            `public:"View"`,
			wantVisibility: "view",
		},
		{
			name:           "Public tag with invalid value",
			tag:            `public:"invalid"`,
			wantVisibility: "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structTag := reflect.StructTag(tt.tag)
			gotVisibility := parsePublicTag(structTag)

			if gotVisibility != tt.wantVisibility {
				t.Errorf("parsePublicTag() = %v, want %v", gotVisibility, tt.wantVisibility)
			}
		})
	}
}

// TestParseValidationTags tests parsing validation struct tags
func TestParseValidationTags(t *testing.T) {
	tests := []struct {
		name         string
		tag          string
		wantRequired bool
		wantOptional bool
		wantMin      string
		wantMax      string
	}{
		{
			name:         "Required in binding tag",
			tag:          `binding:"required"`,
			wantRequired: true,
			wantOptional: false,
			wantMin:      "",
			wantMax:      "",
		},
		{
			name:         "Required in validate tag",
			tag:          `validate:"required"`,
			wantRequired: true,
			wantOptional: false,
			wantMin:      "",
			wantMax:      "",
		},
		{
			name:         "Optional in binding tag",
			tag:          `binding:"optional"`,
			wantRequired: false,
			wantOptional: true,
			wantMin:      "",
			wantMax:      "",
		},
		{
			name:         "Optional in validate tag",
			tag:          `validate:"optional"`,
			wantRequired: false,
			wantOptional: true,
			wantMin:      "",
			wantMax:      "",
		},
		{
			name:         "Required with min and max",
			tag:          `validate:"required,min=1,max=100"`,
			wantRequired: true,
			wantOptional: false,
			wantMin:      "1",
			wantMax:      "100",
		},
		{
			name:         "Min and max without required",
			tag:          `validate:"min=0,max=50"`,
			wantRequired: false,
			wantOptional: false,
			wantMin:      "0",
			wantMax:      "50",
		},
		{
			name:         "Binding with multiple validations",
			tag:          `binding:"required,max=10"`,
			wantRequired: true,
			wantOptional: false,
			wantMin:      "",
			wantMax:      "10",
		},
		{
			name:         "Both binding and validate tags",
			tag:          `binding:"required" validate:"min=5,max=100"`,
			wantRequired: true,
			wantOptional: false,
			wantMin:      "5",
			wantMax:      "100",
		},
		{
			name:         "No validation tags",
			tag:          `json:"name"`,
			wantRequired: false,
			wantOptional: false,
			wantMin:      "",
			wantMax:      "",
		},
		{
			name:         "Empty string (no tags)",
			tag:          ``,
			wantRequired: false,
			wantOptional: false,
			wantMin:      "",
			wantMax:      "",
		},
		{
			name:         "Min with gte syntax",
			tag:          `validate:"gte=10"`,
			wantRequired: false,
			wantOptional: false,
			wantMin:      "10",
			wantMax:      "",
		},
		{
			name:         "Max with lte syntax",
			tag:          `validate:"lte=50"`,
			wantRequired: false,
			wantOptional: false,
			wantMin:      "",
			wantMax:      "50",
		},
		{
			name:         "Complex validation",
			tag:          `validate:"required,min=1,max=100,oneof=red green blue"`,
			wantRequired: true,
			wantOptional: false,
			wantMin:      "1",
			wantMax:      "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structTag := reflect.StructTag(tt.tag)
			gotRequired, gotOptional, gotMin, gotMax := parseValidationTags(structTag)

			if gotRequired != tt.wantRequired {
				t.Errorf("parseValidationTags() required = %v, want %v", gotRequired, tt.wantRequired)
			}
			if gotOptional != tt.wantOptional {
				t.Errorf("parseValidationTags() optional = %v, want %v", gotOptional, tt.wantOptional)
			}
			if gotMin != tt.wantMin {
				t.Errorf("parseValidationTags() min = %v, want %v", gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("parseValidationTags() max = %v, want %v", gotMax, tt.wantMax)
			}
		})
	}
}

// TestParseCombinedTags tests parsing all tags together
func TestParseCombinedTags(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		wantInfo TagInfo
	}{
		{
			name: "All tags present - required field",
			tag:  `json:"username" public:"view" validate:"required,min=3,max=20"`,
			wantInfo: TagInfo{
				JSONName:   "username",
				OmitEmpty:  false,
				Ignore:     false,
				Visibility: "view",
				Required:   true,
				Optional:   false,
				Min:        "3",
				Max:        "20",
			},
		},
		{
			name: "JSON with omitempty and optional validation",
			tag:  `json:"email,omitempty" public:"edit" validate:"optional,email"`,
			wantInfo: TagInfo{
				JSONName:   "email",
				OmitEmpty:  true,
				Ignore:     false,
				Visibility: "edit",
				Required:   false,
				Optional:   true,
				Min:        "",
				Max:        "",
			},
		},
		{
			name: "Private field with validation",
			tag:  `json:"password" validate:"required,min=8"`,
			wantInfo: TagInfo{
				JSONName:   "password",
				OmitEmpty:  false,
				Ignore:     false,
				Visibility: "private",
				Required:   true,
				Optional:   false,
				Min:        "8",
				Max:        "",
			},
		},
		{
			name: "Ignored field",
			tag:  `json:"-"`,
			wantInfo: TagInfo{
				JSONName:   "",
				OmitEmpty:  false,
				Ignore:     true,
				Visibility: "private",
				Required:   false,
				Optional:   false,
				Min:        "",
				Max:        "",
			},
		},
		{
			name: "No tags (defaults)",
			tag:  ``,
			wantInfo: TagInfo{
				JSONName:   "",
				OmitEmpty:  false,
				Ignore:     false,
				Visibility: "private",
				Required:   false,
				Optional:   false,
				Min:        "",
				Max:        "",
			},
		},
		{
			name: "Only JSON tag",
			tag:  `json:"field_name,omitempty"`,
			wantInfo: TagInfo{
				JSONName:   "field_name",
				OmitEmpty:  true,
				Ignore:     false,
				Visibility: "private",
				Required:   false,
				Optional:   false,
				Min:        "",
				Max:        "",
			},
		},
		{
			name: "Only public tag",
			tag:  `public:"view"`,
			wantInfo: TagInfo{
				JSONName:   "",
				OmitEmpty:  false,
				Ignore:     false,
				Visibility: "view",
				Required:   false,
				Optional:   false,
				Min:        "",
				Max:        "",
			},
		},
		{
			name: "Only validation tag",
			tag:  `validate:"required"`,
			wantInfo: TagInfo{
				JSONName:   "",
				OmitEmpty:  false,
				Ignore:     false,
				Visibility: "private",
				Required:   true,
				Optional:   false,
				Min:        "",
				Max:        "",
			},
		},
		{
			name: "Binding tag instead of validate",
			tag:  `json:"count" binding:"required,max=100"`,
			wantInfo: TagInfo{
				JSONName:   "count",
				OmitEmpty:  false,
				Ignore:     false,
				Visibility: "private",
				Required:   true,
				Optional:   false,
				Min:        "",
				Max:        "100",
			},
		},
		{
			name: "Complex combination with all features",
			tag:  `json:"user_id,omitempty" public:"edit" binding:"required" validate:"min=1"`,
			wantInfo: TagInfo{
				JSONName:   "user_id",
				OmitEmpty:  true,
				Ignore:     false,
				Visibility: "edit",
				Required:   true,
				Optional:   false,
				Min:        "1",
				Max:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structTag := reflect.StructTag(tt.tag)
			gotInfo := parseCombinedTags(structTag)

			if gotInfo.JSONName != tt.wantInfo.JSONName {
				t.Errorf("parseCombinedTags() JSONName = %v, want %v", gotInfo.JSONName, tt.wantInfo.JSONName)
			}
			if gotInfo.OmitEmpty != tt.wantInfo.OmitEmpty {
				t.Errorf("parseCombinedTags() OmitEmpty = %v, want %v", gotInfo.OmitEmpty, tt.wantInfo.OmitEmpty)
			}
			if gotInfo.Ignore != tt.wantInfo.Ignore {
				t.Errorf("parseCombinedTags() Ignore = %v, want %v", gotInfo.Ignore, tt.wantInfo.Ignore)
			}
			if gotInfo.Visibility != tt.wantInfo.Visibility {
				t.Errorf("parseCombinedTags() Visibility = %v, want %v", gotInfo.Visibility, tt.wantInfo.Visibility)
			}
			if gotInfo.Required != tt.wantInfo.Required {
				t.Errorf("parseCombinedTags() Required = %v, want %v", gotInfo.Required, tt.wantInfo.Required)
			}
			if gotInfo.Optional != tt.wantInfo.Optional {
				t.Errorf("parseCombinedTags() Optional = %v, want %v", gotInfo.Optional, tt.wantInfo.Optional)
			}
			if gotInfo.Min != tt.wantInfo.Min {
				t.Errorf("parseCombinedTags() Min = %v, want %v", gotInfo.Min, tt.wantInfo.Min)
			}
			if gotInfo.Max != tt.wantInfo.Max {
				t.Errorf("parseCombinedTags() Max = %v, want %v", gotInfo.Max, tt.wantInfo.Max)
			}
		})
	}
}

// TestIsSwaggerIgnore tests detecting swaggerignore tags
func TestIsSwaggerIgnore(t *testing.T) {
	tests := []struct {
		name       string
		tag        string
		wantIgnore bool
	}{
		{
			name:       "SwaggerIgnore true",
			tag:        `swaggerignore:"true"`,
			wantIgnore: true,
		},
		{
			name:       "SwaggerIgnore True (capital)",
			tag:        `swaggerignore:"True"`,
			wantIgnore: true,
		},
		{
			name:       "SwaggerIgnore TRUE (all caps)",
			tag:        `swaggerignore:"TRUE"`,
			wantIgnore: true,
		},
		{
			name:       "SwaggerIgnore false",
			tag:        `swaggerignore:"false"`,
			wantIgnore: false,
		},
		{
			name:       "No swaggerignore tag",
			tag:        `json:"name"`,
			wantIgnore: false,
		},
		{
			name:       "Empty swaggerignore tag",
			tag:        `swaggerignore:""`,
			wantIgnore: false,
		},
		{
			name:       "SwaggerIgnore with spaces",
			tag:        `swaggerignore:" true "`,
			wantIgnore: true,
		},
		{
			name:       "Multiple tags with swaggerignore",
			tag:        `json:"name" swaggerignore:"true" validate:"required"`,
			wantIgnore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structTag := reflect.StructTag(tt.tag)
			gotIgnore := isSwaggerIgnore(structTag)

			if gotIgnore != tt.wantIgnore {
				t.Errorf("isSwaggerIgnore() = %v, want %v", gotIgnore, tt.wantIgnore)
			}
		})
	}
}

// TestExtractEnumValues tests extracting enum values from validation tags
func TestExtractEnumValues(t *testing.T) {
	tests := []struct {
		name       string
		tag        string
		wantValues []string
	}{
		{
			name:       "OneOf with simple values",
			tag:        `validate:"oneof=red green blue"`,
			wantValues: []string{"red", "green", "blue"},
		},
		{
			name:       "OneOf with quoted values",
			tag:        `validate:"oneof='value 1' 'value 2'"`,
			wantValues: []string{"value 1", "value 2"},
		},
		{
			name:       "OneOf with numeric values",
			tag:        `validate:"oneof=1 2 3 4 5"`,
			wantValues: []string{"1", "2", "3", "4", "5"},
		},
		{
			name:       "No oneof tag",
			tag:        `validate:"required"`,
			wantValues: nil,
		},
		{
			name:       "Empty oneof",
			tag:        `validate:"oneof="`,
			wantValues: nil,
		},
		{
			name:       "OneOf with mixed values",
			tag:        `validate:"required,oneof=active inactive pending"`,
			wantValues: []string{"active", "inactive", "pending"},
		},
		{
			name:       "Multiple validation rules",
			tag:        `validate:"required,min=1,oneof=small medium large,max=10"`,
			wantValues: []string{"small", "medium", "large"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structTag := reflect.StructTag(tt.tag)
			gotValues := extractEnumValues(structTag)

			if len(gotValues) != len(tt.wantValues) {
				t.Errorf("extractEnumValues() length = %v, want %v", len(gotValues), len(tt.wantValues))
				return
			}

			for i, val := range gotValues {
				if val != tt.wantValues[i] {
					t.Errorf("extractEnumValues()[%d] = %v, want %v", i, val, tt.wantValues[i])
				}
			}
		})
	}
}
