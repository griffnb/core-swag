package orchestrator

import (
	"go/ast"
	"testing"

	"github.com/griffnb/core-swag/internal/domain"
	"github.com/griffnb/core-swag/internal/registry"
)

func TestRegistryNameResolver_UniqueType(t *testing.T) {
	reg := registry.NewService()

	// Register a unique type (not NotUnique)
	typeSpec := &ast.TypeSpec{Name: ast.NewIdent("Role")}
	typeDef := &domain.TypeSpecDef{
		TypeSpec: typeSpec,
		PkgPath:  "github.com/user/project/internal/constants",
		File:     &ast.File{Name: ast.NewIdent("constants")},
	}
	reg.AddTypeSpecForTest("constants.Role", typeDef)

	resolver := newRegistryNameResolver(reg)
	result := resolver.ResolveDefinitionName("github.com/user/project/internal/constants.Role")

	if result != "constants.Role" {
		t.Errorf("expected short name 'constants.Role', got '%s'", result)
	}
}

func TestRegistryNameResolver_NotUniqueType(t *testing.T) {
	reg := registry.NewService()

	// Register a NotUnique type (full-path key)
	typeSpec := &ast.TypeSpec{Name: ast.NewIdent("Source")}
	typeDef := &domain.TypeSpecDef{
		TypeSpec:  typeSpec,
		PkgPath:   "github.com/chargebee/chargebee-go/v3/enum",
		NotUnique: true,
		File:      &ast.File{Name: ast.NewIdent("enum")},
	}
	fullPathKey := "github_com_chargebee_chargebee-go_v3_enum.Source"
	reg.AddTypeSpecForTest(fullPathKey, typeDef)

	resolver := newRegistryNameResolver(reg)
	result := resolver.ResolveDefinitionName("github.com/chargebee/chargebee-go/v3/enum.Source")

	if result != fullPathKey {
		t.Errorf("expected full-path name '%s', got '%s'", fullPathKey, result)
	}
}

func TestRegistryNameResolver_NotInRegistry(t *testing.T) {
	reg := registry.NewService()
	resolver := newRegistryNameResolver(reg)

	result := resolver.ResolveDefinitionName("github.com/user/project/internal/unknown.Type")

	if result != "unknown.Type" {
		t.Errorf("expected fallback short name 'unknown.Type', got '%s'", result)
	}
}

func TestRegistryNameResolver_IsNotUnique(t *testing.T) {
	reg := registry.NewService()

	// NotUnique type registered under its full-path key.
	notUnique := &domain.TypeSpecDef{
		TypeSpec:  &ast.TypeSpec{Name: ast.NewIdent("Source")},
		PkgPath:   "github.com/chargebee/chargebee-go/v3/enum",
		NotUnique: true,
		File:      &ast.File{Name: ast.NewIdent("enum")},
	}
	reg.AddTypeSpecForTest("github_com_chargebee_chargebee-go_v3_enum.Source", notUnique)

	// Unique type registered under its short key.
	unique := &domain.TypeSpecDef{
		TypeSpec: &ast.TypeSpec{Name: ast.NewIdent("Role")},
		PkgPath:  "github.com/user/project/internal/constants",
		File:     &ast.File{Name: ast.NewIdent("constants")},
	}
	reg.AddTypeSpecForTest("constants.Role", unique)

	resolver := newRegistryNameResolver(reg)

	if !resolver.IsNotUnique("github.com/chargebee/chargebee-go/v3/enum.Source") {
		t.Error("expected enum.Source to be reported NotUnique")
	}
	if resolver.IsNotUnique("github.com/user/project/internal/constants.Role") {
		t.Error("expected constants.Role to be reported unique")
	}
	if resolver.IsNotUnique("github.com/user/project/internal/missing.Type") {
		t.Error("expected unknown type to be reported unique (false)")
	}
}

func TestExtractShortTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"github.com/user/project/internal/constants.Role", "constants.Role"},
		{"github.com/chargebee/chargebee-go/v3/enum.Source", "enum.Source"},
		{"constants.Role", "constants.Role"},
		{"Role", "Role"},
	}

	for _, tt := range tests {
		result := extractShortTypeName(tt.input)
		if result != tt.expected {
			t.Errorf("extractShortTypeName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMakeFullPathDefName2(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"github.com/chargebee/chargebee-go/v3/enum.Source",
			"github_com_chargebee_chargebee-go_v3_enum.Source",
		},
		{
			"github.com/user/project/internal/constants.Role",
			"github_com_user_project_internal_constants.Role",
		},
		{"Role", "Role"}, // no dot
	}

	for _, tt := range tests {
		result := makeFullPathDefName2(tt.input)
		if result != tt.expected {
			t.Errorf("makeFullPathDefName2(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Verify an empty registry returns the short name fallback.
func TestRegistryNameResolver_EmptyRegistry(t *testing.T) {
	resolver := newRegistryNameResolver(registry.NewService())
	result := resolver.ResolveDefinitionName("github.com/user/project/internal/types.Missing")
	if result != "types.Missing" {
		t.Errorf("expected 'types.Missing', got '%s'", result)
	}
}
