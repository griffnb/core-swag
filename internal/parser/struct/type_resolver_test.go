package structparser

import (
	"testing"
)

// TestSplitGenericTypeName tests splitting generic type names into base type and parameters
func TestSplitGenericTypeName(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantBaseName   string
		wantParams     []string
		shouldSucceed  bool
	}{
		{
			name:          "Simple generic with one parameter",
			input:         "fields.StructField[string]",
			wantBaseName:  "fields.StructField",
			wantParams:    []string{"string"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with pointer parameter",
			input:         "fields.StructField[*model.Account]",
			wantBaseName:  "fields.StructField",
			wantParams:    []string{"*model.Account"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with slice parameter",
			input:         "fields.StructField[[]string]",
			wantBaseName:  "fields.StructField",
			wantParams:    []string{"[]string"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with slice of pointers",
			input:         "fields.StructField[[]*model.User]",
			wantBaseName:  "fields.StructField",
			wantParams:    []string{"[]*model.User"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with map parameter",
			input:         "fields.StructField[map[string]int]",
			wantBaseName:  "fields.StructField",
			wantParams:    []string{"map[string]int"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with nested generic",
			input:         "Wrapper[Inner[int]]",
			wantBaseName:  "Wrapper",
			wantParams:    []string{"Inner[int]"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with multiple parameters",
			input:         "Map[string,int]",
			wantBaseName:  "Map",
			wantParams:    []string{"string", "int"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with spaces (should be removed)",
			input:         "fields.StructField[ string ]",
			wantBaseName:  "fields.StructField",
			wantParams:    []string{"string"},
			shouldSucceed: true,
		},
		{
			name:          "Generic with complex nested type",
			input:         "Response[map[string][]model.User]",
			wantBaseName:  "Response",
			wantParams:    []string{"map[string][]model.User"},
			shouldSucceed: true,
		},
		{
			name:          "Not a generic type",
			input:         "string",
			wantBaseName:  "",
			wantParams:    nil,
			shouldSucceed: false,
		},
		{
			name:          "Missing closing bracket",
			input:         "fields.StructField[string",
			wantBaseName:  "",
			wantParams:    nil,
			shouldSucceed: false,
		},
		{
			name:          "Empty generic",
			input:         "fields.StructField[]",
			wantBaseName:  "fields.StructField",
			wantParams:    []string{},
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseName, params := splitGenericTypeName(tt.input)

			if tt.shouldSucceed {
				if baseName != tt.wantBaseName {
					t.Errorf("splitGenericTypeName() baseName = %v, want %v", baseName, tt.wantBaseName)
				}
				if len(params) != len(tt.wantParams) {
					t.Errorf("splitGenericTypeName() params length = %v, want %v", len(params), len(tt.wantParams))
					return
				}
				for i, param := range params {
					if param != tt.wantParams[i] {
						t.Errorf("splitGenericTypeName() params[%d] = %v, want %v", i, param, tt.wantParams[i])
					}
				}
			} else {
				if baseName != "" || params != nil {
					t.Errorf("splitGenericTypeName() should fail but got baseName=%v, params=%v", baseName, params)
				}
			}
		})
	}
}

// TestExtractInnerType tests extracting the inner type from generic wrappers
func TestExtractInnerType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantError bool
	}{
		{
			name:      "Extract string from StructField",
			input:     "fields.StructField[string]",
			wantType:  "string",
			wantError: false,
		},
		{
			name:      "Extract int64 from StructField",
			input:     "fields.StructField[int64]",
			wantType:  "int64",
			wantError: false,
		},
		{
			name:      "Extract bool from StructField",
			input:     "fields.StructField[bool]",
			wantType:  "bool",
			wantError: false,
		},
		{
			name:      "Extract model pointer from StructField",
			input:     "fields.StructField[*model.Account]",
			wantType:  "model.Account", // Should strip pointer
			wantError: false,
		},
		{
			name:      "Extract slice from StructField",
			input:     "fields.StructField[[]string]",
			wantType:  "[]string",
			wantError: false,
		},
		{
			name:      "Extract slice of pointers from StructField",
			input:     "fields.StructField[[]*model.User]",
			wantType:  "[]model.User", // Should strip pointers from slice elements
			wantError: false,
		},
		{
			name:      "Extract map from StructField",
			input:     "fields.StructField[map[string]int]",
			wantType:  "map[string]int",
			wantError: false,
		},
		{
			name:      "Extract nested generic",
			input:     "Wrapper[Inner[int]]",
			wantType:  "Inner[int]",
			wantError: false,
		},
		{
			name:      "Non-generic type should return as-is",
			input:     "string",
			wantType:  "string",
			wantError: false,
		},
		{
			name:      "Model type should return as-is",
			input:     "model.Account",
			wantType:  "model.Account",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := extractInnerType(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("extractInnerType() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("extractInnerType() unexpected error: %v", err)
				return
			}

			if gotType != tt.wantType {
				t.Errorf("extractInnerType() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

// TestIsCustomModel tests detecting custom model types like fields.StructField
func TestIsCustomModel(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{
			name:     "fields.StructField is custom model",
			typeName: "fields.StructField",
			want:     true,
		},
		{
			name:     "fields.StructField with generic is custom model",
			typeName: "fields.StructField[string]",
			want:     true,
		},
		{
			name:     "string is not custom model",
			typeName: "string",
			want:     false,
		},
		{
			name:     "int64 is not custom model",
			typeName: "int64",
			want:     false,
		},
		{
			name:     "bool is not custom model",
			typeName: "bool",
			want:     false,
		},
		{
			name:     "model.Account is not custom model",
			typeName: "model.Account",
			want:     false,
		},
		{
			name:     "[]string is not custom model",
			typeName: "[]string",
			want:     false,
		},
		{
			name:     "map[string]int is not custom model",
			typeName: "map[string]int",
			want:     false,
		},
		{
			name:     "Empty string is not custom model",
			typeName: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCustomModel(tt.typeName)
			if got != tt.want {
				t.Errorf("isCustomModel(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

// TestStripPointer tests removing pointer prefix from type names
func TestStripPointer(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     string
	}{
		{
			name:     "Strip pointer from simple type",
			typeName: "*string",
			want:     "string",
		},
		{
			name:     "Strip pointer from model",
			typeName: "*model.Account",
			want:     "model.Account",
		},
		{
			name:     "Strip pointer from qualified name",
			typeName: "*github.com/user/pkg.Type",
			want:     "github.com/user/pkg.Type",
		},
		{
			name:     "Non-pointer type returns as-is",
			typeName: "string",
			want:     "string",
		},
		{
			name:     "Non-pointer model returns as-is",
			typeName: "model.Account",
			want:     "model.Account",
		},
		{
			name:     "Empty string returns empty",
			typeName: "",
			want:     "",
		},
		{
			name:     "Multiple pointers (double pointer)",
			typeName: "**string",
			want:     "string", // Should strip all leading asterisks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripPointer(tt.typeName)
			if got != tt.want {
				t.Errorf("stripPointer(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

// TestNormalizeGenericTypeName tests normalizing type names for use as identifiers
func TestNormalizeGenericTypeName(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     string
	}{
		{
			name:     "Replace dots with underscores",
			typeName: "model.Account",
			want:     "model_Account",
		},
		{
			name:     "Replace multiple dots",
			typeName: "github.com/user/pkg.Type",
			want:     "github_com/user/pkg_Type",
		},
		{
			name:     "No dots returns as-is",
			typeName: "string",
			want:     "string",
		},
		{
			name:     "Complex generic type",
			typeName: "map.Map[string.String,int.Int]",
			want:     "map_Map[string_String,int_Int]",
		},
		{
			name:     "Empty string returns empty",
			typeName: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeGenericTypeName(tt.typeName)
			if got != tt.want {
				t.Errorf("normalizeGenericTypeName(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

// TestIsSliceType tests detecting slice types
func TestIsSliceType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{
			name:     "Simple slice",
			typeName: "[]string",
			want:     true,
		},
		{
			name:     "Slice of models",
			typeName: "[]model.Account",
			want:     true,
		},
		{
			name:     "Slice of pointers",
			typeName: "[]*model.User",
			want:     true,
		},
		{
			name:     "Multi-dimensional slice",
			typeName: "[][]string",
			want:     true,
		},
		{
			name:     "Not a slice - simple type",
			typeName: "string",
			want:     false,
		},
		{
			name:     "Not a slice - array with size",
			typeName: "[5]string",
			want:     false,
		},
		{
			name:     "Not a slice - map",
			typeName: "map[string]int",
			want:     false,
		},
		{
			name:     "Empty string",
			typeName: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSliceType(tt.typeName)
			if got != tt.want {
				t.Errorf("isSliceType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

// TestIsMapType tests detecting map types
func TestIsMapType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{
			name:     "Simple map",
			typeName: "map[string]int",
			want:     true,
		},
		{
			name:     "Map with model value",
			typeName: "map[string]model.Account",
			want:     true,
		},
		{
			name:     "Map with slice value",
			typeName: "map[string][]int",
			want:     true,
		},
		{
			name:     "Nested map",
			typeName: "map[string]map[string]int",
			want:     true,
		},
		{
			name:     "Not a map - simple type",
			typeName: "string",
			want:     false,
		},
		{
			name:     "Not a map - slice",
			typeName: "[]string",
			want:     false,
		},
		{
			name:     "Empty string",
			typeName: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMapType(tt.typeName)
			if got != tt.want {
				t.Errorf("isMapType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

// TestGetSliceElementType tests extracting element type from slice
func TestGetSliceElementType(t *testing.T) {
	tests := []struct {
		name      string
		typeName  string
		wantType  string
		wantError bool
	}{
		{
			name:      "Simple slice",
			typeName:  "[]string",
			wantType:  "string",
			wantError: false,
		},
		{
			name:      "Slice of models",
			typeName:  "[]model.Account",
			wantType:  "model.Account",
			wantError: false,
		},
		{
			name:      "Slice of pointers",
			typeName:  "[]*model.User",
			wantType:  "*model.User",
			wantError: false,
		},
		{
			name:      "Multi-dimensional slice",
			typeName:  "[][]string",
			wantType:  "[]string",
			wantError: false,
		},
		{
			name:      "Not a slice",
			typeName:  "string",
			wantType:  "",
			wantError: true,
		},
		{
			name:      "Empty slice notation",
			typeName:  "[]",
			wantType:  "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := getSliceElementType(tt.typeName)

			if tt.wantError {
				if err == nil {
					t.Errorf("getSliceElementType() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("getSliceElementType() unexpected error: %v", err)
				return
			}

			if gotType != tt.wantType {
				t.Errorf("getSliceElementType(%q) = %v, want %v", tt.typeName, gotType, tt.wantType)
			}
		})
	}
}
